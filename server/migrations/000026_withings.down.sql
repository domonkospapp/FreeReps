DELETE FROM metric_allowlist WHERE metric_name IN (
    'fat_mass', 'muscle_mass', 'bone_mass', 'body_water', 'blood_pressure_heart_rate'
);
DROP TABLE IF EXISTS withings_sync_state;
DROP TABLE IF EXISTS withings_tokens;
