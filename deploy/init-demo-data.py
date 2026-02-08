#!/usr/bin/env python3

import sqlite3
import sys

TRANSLATIONS = {
    "join_success": "Welcome to the roulette! You're in the game now.",
    "leave_success": "%s has left the roulette.",
    "leave_not_in_game": "You're not in the game yet.",
    "no_participants": "No players registered yet. Use /join to enter the roulette!",
    "already_played": "The wheel has already been spun today! Today's winner is %s!",
    "fallback_winner": "And the winner is... %s!",
    "stats_header": "<b>Hall of Fame:</b>",
    "stats_year_header": "<b>Hall of Fame (%d):</b>",
    "stats_invalid_year": "Invalid year: %s",
    "stats_no_results": "No results for %d.",
    "stats_line": "%d. %s — %d win(s)",
    "participants_header": "<b>Players in the roulette:</b>",
    "reset_no_result": "Nothing to reset. The wheel hasn't been spun yet.",
    "reset_success": "The wheel has been reset. Spin again with /roll!",
    "unknown_user": "Player #%d",
}

MESSAGE_SETS = {
    1: [
        "Spinning the wheel...",
        "Round and round it goes...",
        "Almost there...",
    ],
    2: [
        "The roulette is starting!",
        "Who will it be today?",
        "Drumroll please...",
        "And the chosen one is...",
    ],
    3: [
        "Let's find today's lucky winner!",
        "Scanning participants...",
        "Target acquired!",
    ],
}


def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <database>", file=sys.stderr)
        sys.exit(1)

    db = sqlite3.connect(sys.argv[1])
    cur = db.cursor()

    for key, value in TRANSLATIONS.items():
        cur.execute(
            "INSERT OR IGNORE INTO translations (key, value) VALUES (?, ?)",
            (key, value),
        )

    for set_id, messages in MESSAGE_SETS.items():
        cur.execute("INSERT OR IGNORE INTO message_sets (id) VALUES (?)", (set_id,))
        for position, body in enumerate(messages, start=1):
            cur.execute(
                "INSERT OR IGNORE INTO set_messages (set_id, position, body) VALUES (?, ?, ?)",
                (set_id, position, body),
            )

    db.commit()
    db.close()


if __name__ == "__main__":
    main()
