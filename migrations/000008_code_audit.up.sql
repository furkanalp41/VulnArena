-- Add line-targeting and CVE reference fields for Code Audit mode
ALTER TABLE challenges
  ADD COLUMN vulnerable_lines INTEGER[] DEFAULT '{}',
  ADD COLUMN cve_reference VARCHAR(100);

-- Submissions now include user-targeted line numbers
ALTER TABLE submissions
  ADD COLUMN target_lines INTEGER[] DEFAULT '{}';
