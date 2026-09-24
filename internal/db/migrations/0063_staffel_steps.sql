-- HAUSV-787: structured contractual steps inherit index_clauses tenant scope.
ALTER TABLE index_clauses ADD COLUMN staffel_steps TEXT NOT NULL DEFAULT '[]'
    CHECK (json_valid(staffel_steps) AND json_type(staffel_steps) = 'array');
