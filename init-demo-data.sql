-- Default translations
INSERT OR IGNORE INTO translations (key, value) VALUES ('join_success', 'Welcome to the roulette! You''re in the game now.');
INSERT OR IGNORE INTO translations (key, value) VALUES ('leave_success', '%s has left the roulette.');
INSERT OR IGNORE INTO translations (key, value) VALUES ('leave_not_in_game', 'You''re not in the game yet.');
INSERT OR IGNORE INTO translations (key, value) VALUES ('no_participants', 'No players registered yet. Use /join to enter the roulette!');
INSERT OR IGNORE INTO translations (key, value) VALUES ('already_played', 'The wheel has already been spun today! Today''s winner is %s!');
INSERT OR IGNORE INTO translations (key, value) VALUES ('fallback_winner', 'And the winner is... %s!');
INSERT OR IGNORE INTO translations (key, value) VALUES ('stats_header', '<b>Hall of Fame:</b>');
INSERT OR IGNORE INTO translations (key, value) VALUES ('stats_year_header', '<b>Hall of Fame (%d):</b>');
INSERT OR IGNORE INTO translations (key, value) VALUES ('stats_invalid_year', 'Invalid year: %s');
INSERT OR IGNORE INTO translations (key, value) VALUES ('stats_no_results', 'No results for %d.');
INSERT OR IGNORE INTO translations (key, value) VALUES ('stats_line', '%d. %s — %d win(s)');
INSERT OR IGNORE INTO translations (key, value) VALUES ('participants_header', '<b>Players in the roulette:</b>');
INSERT OR IGNORE INTO translations (key, value) VALUES ('reset_no_result', 'Nothing to reset. The wheel hasn''t been spun yet.');
INSERT OR IGNORE INTO translations (key, value) VALUES ('reset_success', 'The wheel has been reset. Spin again with /roll!');
INSERT OR IGNORE INTO translations (key, value) VALUES ('unknown_user', 'Player #%d');

-- Message sets
INSERT OR IGNORE INTO message_sets (id) VALUES (1);
INSERT OR IGNORE INTO message_sets (id) VALUES (2);
INSERT OR IGNORE INTO message_sets (id) VALUES (3);

INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (1, 1, 'Spinning the wheel...');
INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (1, 2, 'Round and round it goes...');
INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (1, 3, 'Almost there...');

INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (2, 1, 'The roulette is starting!');
INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (2, 2, 'Who will it be today?');
INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (2, 3, 'Drumroll please...');
INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (2, 4, 'And the chosen one is...');

INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (3, 1, 'Let''s find today''s lucky winner!');
INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (3, 2, 'Scanning participants...');
INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (3, 3, 'Target acquired!');
