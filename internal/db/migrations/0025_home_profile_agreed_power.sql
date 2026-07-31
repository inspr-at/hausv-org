-- Vereinbarte Anschlussleistung des Hausanschlusses in kW.
--
-- Ab 01.01.2027 bemisst der Entwurf der SNE-G-V die verrechnete Leistung mit
-- mindestens 20 % dieser vereinbarten Leistung, jedenfalls mindestens 2 kW.
-- Ohne den Wert konnte die Mindestbemessung nie greifen: die Schätzung lief
-- bisher fest mit 0 und sah damit nur die gemessene Monatsspitze.
--
-- NULL bedeutet ausdrücklich "nicht erfasst" und nicht "null kW" — nur so
-- bleibt die Mindestbemessung stumm, solange der Wert unbekannt ist.
ALTER TABLE home_profiles
    ADD COLUMN agreed_power_kw REAL;
