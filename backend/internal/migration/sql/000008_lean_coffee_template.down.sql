-- Delete phase timers for Lean Coffee template
DELETE FROM template_phase_timers
WHERE template_id IN (SELECT id FROM templates WHERE name = 'Lean Coffee' AND is_built_in = true);

-- Delete Lean Coffee template
DELETE FROM templates WHERE name = 'Lean Coffee' AND is_built_in = true;
