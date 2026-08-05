-- Map the Alpha Progression exercise names onto Hevy's catalog.
--
-- Muscle groups exist only on catalog entries. The 6922 sets imported from Alpha
-- Progression carry a free-text exercise name and nothing else, so without this
-- translation every per-muscle figure would start at the first Hevy workout and
-- report nothing about the training logged since March 2025.
--
-- What is decided here is only which catalog entry a name refers to. Which
-- muscle that entry works is Hevy's statement, not one made in this repository.
--
-- note = 'approximate' marks a name Hevy has no exact counterpart for, where the
-- nearest movement was chosen. 22 names, 918 sets. Almost all of them sit on one
-- axis — upper_back against lats for the cable and machine rows — so the total
-- amount of back work is unaffected while its split between the two is not
-- firm. GetTrainingVolume reports the approximate share per muscle group so a
-- reader can tell how far a figure can be pushed.
--
-- A NULL template id records a name that was examined and has no counterpart,
-- which is a different statement from a name nobody has looked at yet.
--
-- The WHERE EXISTS guard matters: exercise_name_map references
-- exercise_templates, and on a fresh instance the catalog is empty until the
-- first Hevy sync. Such an instance has no Alpha history either, so an empty
-- mapping is the correct outcome rather than a failed migration.
INSERT INTO exercise_name_map (source, exercise_name, exercise_template_id, note)
SELECT 'Alpha Progression', v.name, v.template_id, v.note
FROM (VALUES
    ('Bench Press', '79D0BB3A', ''),  -- Bench Press (Barbell) [chest]
    ('Lateral Raises', '422B08F1', ''),  -- Lateral Raise (Dumbbell) [shoulders]
    ('Seated Shoulder Press', 'B09A1304', ''),  -- Overhead Press (Smith Machine) [shoulders]
    ('Incline Bench Press', '50DFDFAB', ''),  -- Incline Bench Press (Barbell) [chest]
    ('Chest Press', '7EB3F7C3', ''),  -- Chest Press (Machine) [chest]
    ('Standing Calf Raises', 'E05C2C38', ''),  -- Standing Calf Raise (Machine) [calves]
    ('Chin-Ups', '29083183', ''),  -- Chin Up [lats]
    ('Curls', '01A35BF9', ''),  -- EZ Bar Biceps Curl [biceps]
    ('Leg Press', 'C7973E0E', ''),  -- Leg Press (Machine) [quadriceps]
    ('Hanging Leg Raises', 'F8356514', ''),  -- Hanging Leg Raise [abdominals]
    ('Reverse Butterfly with Wide Grip', 'D8281C62', ''),  -- Rear Delt Reverse Fly (Machine) [shoulders]
    ('Rows with Close Grip', '0393F233', 'approximate'),  -- Seated Cable Row - V Grip (Cable) [upper_back]
    ('Romanian Deadlifts', '72CFFAD5', ''),  -- Romanian Deadlift (Dumbbell) [hamstrings]
    ('Decline Crunches', 'BC10A922', ''),  -- Decline Crunch [abdominals]
    ('Leg Extensions', '75A4F6C4', ''),  -- Leg Extension (Machine) [quadriceps]
    ('Reverse Flys', 'C315DC2A', ''),  -- Rear Delt Reverse Fly (Cable) [shoulders]
    ('Rows with Wide Neutral Grip', 'C3BCABB3', 'approximate'),  -- Seated Cable Row - Bar Wide Grip [upper_back]
    ('Lat Pulldowns with Wide Overhand Grip', '6A6C31A5', ''),  -- Lat Pulldown (Cable) [lats]
    ('One-Arm Rows', 'F1E57334', ''),  -- Dumbbell Row [lats]
    ('Reverse Lunges', 'FFDA283B', ''),  -- Reverse Lunge (Dumbbell) [quadriceps]
    ('Hyperextensions on Roman Chair', '4F5866F8', ''),  -- Back Extension (Hyperextension) [lower_back]
    ('Lying Leg Curls', 'B8127AD1', ''),  -- Lying Leg Curl (Machine) [hamstrings]
    ('Triceps Pushdowns with Rope', '94B7239B', ''),  -- Triceps Rope Pushdown [triceps]
    ('Pullovers with Rope', '9273BA17', 'approximate'),  -- Rope Straight Arm Pulldown [lats]
    ('Overhead Triceps Extensions with Rope', 'B5EFBF9C', ''),  -- Overhead Triceps Extension (Cable) [triceps]
    ('Decline Bench Press', 'DA0F0470', ''),  -- Decline Bench Press (Barbell) [chest]
    ('Incline Curls', '8BAB2735', ''),  -- Seated Incline Curl (Dumbbell) [biceps]
    ('Rows with Reverse Grip', 'F1D60854', 'approximate'),  -- Seated Cable Row - Bar Grip [upper_back]
    ('Bulgarian Split Squats', 'B5D3A742', ''),  -- Bulgarian Split Squat (Dumbbell) [quadriceps]
    ('Bent-Over Rows', '55E6546F', ''),  -- Bent Over Row (Barbell) [upper_back]
    ('Skull Crushers', '875F585F', ''),  -- Skullcrusher (Barbell) [triceps]
    ('Sumo Leg Press', 'C7973E0E', 'approximate'),  -- Leg Press (Machine) [quadriceps]
    ('Seated Shoulder Press with Close Grip', '9930DF71', 'approximate'),  -- Seated Overhead Press (Dumbbell) [shoulders]
    ('Incline Reverse Flys', 'B582299E', ''),  -- Chest Supported Reverse Fly (Dumbbell) [shoulders]
    ('One-Arm Lateral Raises at Hip Height', 'BE289E45', ''),  -- Lateral Raise (Cable) [shoulders]
    ('Walking Lunges', 'A733CC5B', ''),  -- Walking Lunge (Dumbbell) [quadriceps]
    ('Hack Squats', '1E42FD5F', ''),  -- Hack Squat (Machine) [quadriceps]
    ('Chest-Supported Rows with Wide Grip', 'AA1EB7D8', 'approximate'),  -- Iso-Lateral Row (Machine) [upper_back]
    ('Hammer Curls', '7E3BC8B6', ''),  -- Hammer Curl (Dumbbell) [biceps]
    ('Sumo Squats', 'DDCC3821', 'approximate'),  -- Squat (Smith Machine) [quadriceps]
    ('T-Bar Rows with Close Grip', '6A8D3193', ''),  -- Chest Supported T Bar Row [upper_back]
    ('Pull-Ups with Close Neutral Grip', '1B2B1E7C', ''),  -- Pull Up [lats]
    ('Chest-Supported High Rows with Overhand Grip', 'BC3492DA', ''),  -- Iso-Lateral High Row (Machine) [lats]
    ('Seated Leg Curls', '11A123F3', ''),  -- Seated Leg Curl (Machine) [hamstrings]
    ('One-Arm Rows with Wide Grip', 'F1E57334', 'approximate'),  -- Dumbbell Row [lats]
    ('Crunch Machine', 'EB43ADD4', ''),  -- Crunch (Machine) [abdominals]
    ('Back Extensions on Roman Chair', '4F5866F8', ''),  -- Back Extension (Hyperextension) [lower_back]
    ('Bent-Over Lateral Raises', 'E5988A0A', ''),  -- Rear Delt Reverse Fly (Dumbbell) [shoulders]
    ('Lat Pulldowns with Close Neutral Grip', '4E5257DE', ''),  -- Lat Pulldown - Close Grip (Cable) [lats]
    ('Forward Lunges', 'B537D09F', ''),  -- Lunge (Dumbbell) [quadriceps]
    ('Lat Pulldowns with Reverse Grip', '046E25A2', ''),  -- Reverse Grip Lat Pulldown (Cable) [lats]
    ('Bench Press with Close Grip', '35B51B87', ''),  -- Bench Press - Close Grip (Barbell) [triceps]
    ('Face Pulls with Rope', 'BE640BA0', ''),  -- Face Pull [shoulders]
    ('JM Press', NULL, 'no counterpart in the catalog'),  -- unmapped on purpose
    ('Seated Calf Raises', '062AB91A', ''),  -- Seated Calf Raise [calves]
    ('Dips', '28BB4A95', 'approximate'),  -- Triceps Dip [triceps]
    ('Bench Dips', 'CD6DC8E5', ''),  -- Bench Dip [triceps]
    ('Hanging Leg Raises with Twist', 'F8356514', 'approximate'),  -- Hanging Leg Raise [abdominals]
    ('Pullovers on Bench', '67280085', ''),  -- Pullover (Dumbbell) [lats]
    ('Spider Curls', '90427D4A', ''),  -- Spider Curl (Dumbbell) [biceps]
    ('Front Raises', '8293E554', ''),  -- Front Raise (Dumbbell) [shoulders]
    ('Incline Bench Press 20°', '3A6FA3D1', ''),  -- Incline Bench Press (Smith Machine) [chest]
    ('Incline Rows', '914F3A96', ''),  -- Chest Supported Incline Row (Dumbbell) [upper_back]
    ('Overhead Triceps Extensions', '8347DFD1', 'approximate'),  -- Single Arm Tricep Extension (Dumbbell) [triceps]
    ('Double Crunch Machine Sideways', 'DBE341AA', 'approximate'),  -- Oblique Crunch [abdominals]
    ('One-Arm Lateral Raises', 'BE289E45', ''),  -- Lateral Raise (Cable) [shoulders]
    ('High Cable Flys', '651F844C', ''),  -- Cable Fly Crossovers [chest]
    ('Back Extensions', 'A05C064D', ''),  -- Back Extension (Machine) [lower_back]
    ('Incline Skull Crushers', '68F8A292', 'approximate'),  -- Skullcrusher (Dumbbell) [triceps]
    ('Y Raises', 'F21D5693', 'approximate'),  -- Chest Supported Y Raise (Dumbbell) [shoulders]
    ('Incline Bench Press 60°', '3A6FA3D1', ''),  -- Incline Bench Press (Smith Machine) [chest]
    ('Pendlay Rows with Reverse Grip', '018ADC12', ''),  -- Pendlay Row (Barbell) [upper_back]
    ('Standing Chest Press', 'EAC7D9C5', ''),  -- Chest Press (Band) [chest]
    ('Butterfly with Wide Grip', '78683336', ''),  -- Chest Fly (Machine) [chest]
    ('Shoulder Press', '9237BAD1', ''),  -- Seated Shoulder Press (Machine) [shoulders]
    ('Incline Bench Press with Close Grip', '3A6FA3D1', 'approximate'),  -- Incline Bench Press (Smith Machine) [chest]
    ('Incline Chest Press', 'FBF92739', ''),  -- Incline Chest Press (Machine) [chest]
    ('Pull Aparts', 'E8D86EE8', ''),  -- Band Pullaparts [shoulders]
    ('Shoulder Press with Close Grip', '9237BAD1', 'approximate'),  -- Seated Shoulder Press (Machine) [shoulders]
    ('Flys', '651F844C', ''),  -- Cable Fly Crossovers [chest]
    ('Latziehen einarmig sitzend', '2EE45F81', ''),  -- Single Arm Lat Pulldown [lats]
    ('Chest-Supported Rows with Reverse Grip', 'AA1EB7D8', 'approximate'),  -- Iso-Lateral Row (Machine) [upper_back]
    ('Chest Dips', '6FCD7755', ''),  -- Chest Dip [chest]
    ('Butterfly with Close Grip', '78683336', ''),  -- Chest Fly (Machine) [chest]
    ('Inverted Rows on Smith Machine', '425805F4', ''),  -- Inverted Row [upper_back]
    ('Standing Shoulder Press', '878CD1D0', 'approximate'),  -- Shoulder Press (Dumbbell) [shoulders]
    ('High Band Flys', 'DDB7C19F', ''),  -- Chest Fly (Band) [chest]
    ('Front Squats', '5046D0A9', ''),  -- Front Squat [quadriceps]
    ('Side Crunches', 'DBE341AA', ''),  -- Oblique Crunch [abdominals]
    ('Overhead Curls', '582ADA23', 'approximate'),  -- Overhead Curl (Cable) [biceps]
    ('Step-Ups', '128A2381', ''),  -- Step Up [quadriceps]
    ('Crunches on Exercise Ball', 'DCF3B31B', 'approximate'),  -- Crunch [abdominals]
    ('Pull-Ups with Wide Overhand Grip', '7C50F118', ''),  -- Wide Pull Up [lats]
    ('Leg Raises', '09C9F635', ''),  -- Lying Leg Raise [abdominals]
    ('Squats with Feet Forward', 'DDCC3821', 'approximate')  -- Squat (Smith Machine) [quadriceps]
) AS v(name, template_id, note)
WHERE v.template_id IS NULL
   OR EXISTS (SELECT 1 FROM exercise_templates t WHERE t.id = v.template_id)
ON CONFLICT (source, exercise_name) DO NOTHING;
