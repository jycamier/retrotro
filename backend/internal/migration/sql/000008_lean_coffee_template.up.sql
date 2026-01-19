-- Migration: Add Lean Coffee template
-- Lean Coffee is a structured discussion format with Topics, On Discuss, and Done columns

-- Insert Lean Coffee template
INSERT INTO templates (id, name, description, columns, is_built_in) VALUES
(uuid_generate_v4(), 'Lean Coffee', 'Format Lean Coffee pour discussions structurées', '[
    {"id": "topic", "name": "Topics", "description": "Sujets à discuter", "color": "#f59e0b", "icon": "list", "order": 0},
    {"id": "discussing", "name": "On Discuss", "description": "Sujet en cours de discussion", "color": "#3b82f6", "icon": "message-circle", "order": 1},
    {"id": "done", "name": "Done", "description": "Sujets discutés", "color": "#22c55e", "icon": "check", "order": 2}
]', true);

-- Insert phase timers for Lean Coffee template
INSERT INTO template_phase_timers (template_id, phase, duration_seconds)
SELECT t.id, phase, duration
FROM templates t
CROSS JOIN (
    VALUES
        ('brainstorm'::retro_phase, 300),
        ('group'::retro_phase, 120),
        ('vote'::retro_phase, 120),
        ('discuss'::retro_phase, 1800),
        ('action'::retro_phase, 300)
) AS phases(phase, duration)
WHERE t.name = 'Lean Coffee' AND t.is_built_in = true;
