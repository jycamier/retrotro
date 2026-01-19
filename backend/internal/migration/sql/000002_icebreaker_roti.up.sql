-- Ajouter les phases à l'enum
ALTER TYPE retro_phase ADD VALUE IF NOT EXISTS 'icebreaker' BEFORE 'brainstorm';
ALTER TYPE retro_phase ADD VALUE IF NOT EXISTS 'roti' AFTER 'action';

-- Type mood météo
CREATE TYPE mood_weather AS ENUM ('sunny', 'partly_cloudy', 'cloudy', 'rainy', 'stormy');

-- Table des humeurs icebreaker
CREATE TABLE icebreaker_moods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    retro_id UUID NOT NULL REFERENCES retrospectives(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mood mood_weather NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT icebreaker_moods_unique UNIQUE (retro_id, user_id)
);
CREATE INDEX idx_icebreaker_moods_retro ON icebreaker_moods(retro_id);

-- Table des votes ROTI (1-5)
CREATE TABLE roti_votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    retro_id UUID NOT NULL REFERENCES retrospectives(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT roti_votes_unique UNIQUE (retro_id, user_id)
);
CREATE INDEX idx_roti_votes_retro ON roti_votes(retro_id);

-- Flag pour révéler les résultats ROTI
ALTER TABLE retrospectives ADD COLUMN roti_revealed BOOLEAN DEFAULT false;
