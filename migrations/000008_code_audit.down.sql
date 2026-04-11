ALTER TABLE submissions
  DROP COLUMN IF EXISTS target_lines;

ALTER TABLE challenges
  DROP COLUMN IF EXISTS cve_reference,
  DROP COLUMN IF EXISTS vulnerable_lines;
