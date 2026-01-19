-- Migration: Add 'waiting' phase to retro_phase enum
-- The waiting phase is the first phase after starting a retro, before icebreaker

-- Add 'waiting' as the first value in the enum (before 'icebreaker')
ALTER TYPE retro_phase ADD VALUE IF NOT EXISTS 'waiting' BEFORE 'icebreaker';
