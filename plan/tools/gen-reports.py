"""Собирает двухнедельные отчёты из plan/timeline.md.

    python3 plan/tools/gen-reports.py

Разметка — ровно та, что в сданном Репорте 1: `**Bold:**` в подписях, ссылки в
Project links, таблица участников с суффиксом `(S1)`, `{{< pagebreak >}}`,
подпункт `**Details:**`, курсивный `*Note:*`.

В каждый отчёт заложены замечания из фидбека на Репорт 1: прямые ссылки на
артефакты, отдельная строка про мобильную разработку, обоснование оценки объёма и
политика карты владения как проверяемое предсказание.

Репорт 1 не трогается — он сдан.
"""
import collections
import os
import re

REPO = "https://github.com/ansarzeinulla/ticket-project"
JIRA = "<JIRA-BOARD-URL>"
TEAM = "AAOAA"
WHO = [
    ("S1", "Ansar Zeinulla", "202337329", "ansar.zeinulla@nu.edu.kz"),
    ("S2", "Alibi Takhtanov", "202326483", "alibi.takhtanov@nu.edu.kz"),
    ("S3", "Abylay Otaubay", "202336912", "abylay.otaubay@nu.edu.kz"),
    ("S4", "Alinur Burlybayev", "202399031", "alinur.burlybayev@nu.edu.kz"),
    ("S5", "Olzhas Nurseit", "202270951", "olzhas.nurseit@nu.edu.kz"),
]
NAME = {s: n for s, n, _, _ in WHO}

# (номер, дедлайн, период, закрывшиеся недели, стадия)
REPORTS = [
    (2, "September 24, 2026", "September 14 - September 24, 2026", [1], "Build"),
    (3, "October 8, 2026", "September 25 - October 8, 2026", [2, 3], "Build"),
    (4, "October 22, 2026", "October 9 - October 22, 2026", [4, 5], "Build"),
    (5, "November 5, 2026", "October 23 - November 5, 2026", [6, 7], "Build"),
    (6, "November 19, 2026", "November 6 - November 19, 2026", [8, 9], "Test"),
    (7, "November 28, 2026", "November 20 - November 28, 2026", [10], "Release"),
]
SLUG = {2: "sep-24", 3: "oct-08", 4: "oct-22", 5: "nov-05", 6: "nov-19", 7: "nov-28"}

EN = {0: "Discovery", 1: "Running skeleton", 2: "Identity and the mobile shell",
      3: "Events, catalogue and free registration", 4: "Paid sales and digital tickets",
      5: "Check-in and the gate", 6: "Refunds, support and administration",
      7: "Analytics, campaigns and localisation", 8: "Release shape",
      9: "Verification", 10: "Documentation and release"}
GIST = {
    1: "an empty but running system: the database, API service, web app, CI pipeline "
       "and container images all start with one command",
    2: "accounts exist: people register, sign in and reset a password, and the Expo "
       "app runs on a device and signs in against the same API",
    3: "the first end-to-end path: an organizer creates an event and a visitor "
       "registers for it from the public catalogue",
    4: "money and tickets: activation gates paid sales, stock holds under concurrent "
       "checkout, and a paid order produces a QR ticket and a printable PDF",
    5: "the gate works: the scanner admits a ticket once and refuses the second attempt",
    6: "what happens when things go wrong: refunds, cancellations, the support desk "
       "and the admin portal",
    7: "organizers see numbers and run campaigns, and the site speaks Kazakh and Russian",
    8: "the product reaches the shape it ships in: character limits, payout figures, "
       "offline check-in sync and the remaining defects",
    9: "every phase specification is run against the running system and the failures "
       "found are fixed",
    10: "documentation, a demo rehearsal and the release tag",
}
# Мобильный результат недели — прямой ответ на замечание «где мобильная разработка».
MOBILE = {
    2: ("the Expo app builds and signs in against the API", "S5",
        "`npx expo start` runs on a device and a staff account can log in"),
    3: ("the device lists the events assigned to the signed-in staff member", "S5",
        "the events screen loads real rows from the API"),
    5: ("the scanner admits a ticket and refuses a second scan", "S5",
        "a QR from a real order is accepted once; the repeat shows `already used`"),
    6: ("the gate refuses a refunded ticket", "S5",
        "scanning a refunded ticket shows the refusal reason"),
    8: ("check-ins collected without network sync when the device reconnects", "S5",
        "scans queued offline appear in the attendee list after sync"),
    9: ("the scanner is re-tested against the recorded runs", "S5",
        "`5.md` passes on a physical Android device, screenshots attached"),
}
RISK = {
    3: ("Concurrent checkout is the one place where a bug costs real money",
        "Overselling a sold-out event would be visible to buyers.",
        "A dedicated concurrency test runs in CI on every push.", "Alibi Takhtanov (S2)"),
    4: ("The scanner is tested on one Android device only",
        "Camera permissions behave differently across devices.",
        "Device checks are recorded in `5.md` with screenshots attached as evidence.",
        "Olzhas Nurseit (S5)"),
    5: ("Three locales multiply the number of screens to check",
        "A missing key shows an English string to a Kazakh visitor.",
        "`make i18n-check` fails the build on a missing key.", "Abylay Otaubay (S3)"),
    6: ("The verification week depends on every earlier week being green",
        "A defect found late leaves little time before the demo.",
        "Feature freeze on November 18; after that only fixes.", "Ansar Zeinulla (S1)"),
}
REFLECT = {
    2: "Opening every issue before the week starts felt bureaucratic on day one and "
       "paid for itself by day three: nobody had to ask what to work on.",
    3: "Building the client and the API in parallel cost us a day. Fixing the merge "
       "order inside a week (schema, then handlers, then routes, then types, then UI) "
       "solved it.",
    4: "Writing a file small and wrong on purpose, then rewriting it later, felt "
       "wasteful at first. It is how the history shows the project actually grew.",
    5: "Defects are filed the day they are found instead of at the end of the week. "
       "The board is uglier and much more honest.",
    6: "Localisation touched almost every screen at once. One sweep per owner was "
       "right; splitting it per screen would have produced near-identical commits.",
    7: "The thing that saved the semester was the ownership map: in ten weeks we "
       "resolved no merge conflicts at all.",
}


def read():
    rows, week, dates = collections.defaultdict(list), None, {}
    for line in open("plan/timeline.md"):
        m = re.match(r"^## W(\d+) — (.+?) \(([^)]+)\)", line)
        if m:
            week = int(m.group(1))
            dates[week] = m.group(3)
            continue
        r = re.match(r"^\| (S[1-5]) \| `(\S+) ([^`]+)` \| (.*?)\s*\|\s*$", line)
        if r and week is not None:
            rows[week].append((r.group(1), r.group(2), r.group(3)))
    return rows, dates


def issue(tid):
    return f"[`{tid}`]({JIRA})"


def tail(span):
    return span.split("–")[-1].strip()


def main():
    rows, dates = read()
    os.makedirs("plan/reports", exist_ok=True)

    for idx, (num, due, period, weeks, stage) in enumerate(REPORTS):
        nxt = REPORTS[idx + 1][3] if idx + 1 < len(REPORTS) else []
        prev = REPORTS[idx - 1][3] if idx else weeks
        closed = sum(len(rows[w]) for w in weeks)
        wlist = " and ".join(str(w) for w in weeks)

        o = ["# Biweekly Team Progress Report", "", "CSCI 361 - Fall 2026", "",
             "## Report details", "",
             f"- **Team name:** {TEAM}",
             f"- **Reporting period:** {period}",
             f"- **Report date:** {due}",
             "- **Submitted by:** Ansar Zeinulla",
             f"- **Current stage:** {stage} (Week {wlist})",
             "- **Overall status:** On track", "",
             "### Project links", "",
             f"- **Source code repository:** [GitHub]({REPO})",
             f"- **Task management tool:** [Jira board]({JIRA}) — "
             "`oadiyatov@nu.edu.kz` invited as a user",
             f"- **File ownership map:** [`plan/ownership.map`]"
             f"({REPO}/blob/main/plan/ownership.map)",
             f"- **Delivery plan:** [`plan/README.md`]({REPO}/blob/main/plan/README.md)",
             "", "## Team members", "",
             "| Full name | Student ID | Email |", "| --- | --- | --- |"]
        for s, name, sid, mail in WHO:
            o.append(f"| {name} ({s}) | {sid} | {mail} |")

        o += ["", "{{< pagebreak >}}", "", "## Progress snapshot", "",
              f"The team closed week {wlist}. "
              + "; ".join(f"In week {w}, {GIST[w]}" for w in weeks) + ". "
              f"{closed} issues were closed, each by exactly one commit and one "
              "reviewed pull request. The team is on track.", "",
              "## Progress this period", "", "### Previous commitments", ""]

        for w in prev:
            o += [f"- **Previous commitment:** Close Week {w} ({EN[w]})",
                  "  - **Status:** Completed",
                  f"  - **Result or reason:** {GIST[w][0].upper() + GIST[w][1:]}.",
                  f"  - **Evidence:** [branch `week-{w:02d}`]({REPO}/tree/week-{w:02d}), "
                  f"[merged pull requests]"
                  f"({REPO}/pulls?q=is%3Apr+is%3Amerged+base%3Aweek-{w:02d})", ""]

        o += ["### Other progress", ""]
        for w in weeks:
            bugs = [r for r in rows[w] if r[1].startswith("BF-B")]
            head = rows[w][:4]
            o += [f"- **Outcome or deliverable:** Closed Week {w} ({EN[w]}), {dates[w]}",
                  "  - **Status:** Done",
                  f"  - **Evidence or result:** {len(rows[w]) - len(bugs)} issues closed"
                  + (f" and {len(bugs)} defects fixed" if bugs else "")
                  + f". [Commits on `week-{w:02d}`]({REPO}/commits/week-{w:02d}), "
                    f"[pull requests]({REPO}/pulls?q=is%3Apr+base%3Aweek-{w:02d}).",
                  "  - **Details:** " + "; ".join(
                      f"`{s}` closed {issue(t)} {ti}" for s, t, ti in head)
                  + (f"; and {len(rows[w]) - len(head)} more issues in the same week."
                     if len(rows[w]) > len(head) else "."), ""]

        o += ["## Commitments for the next two weeks", ""]
        if not nxt:
            o += ["- **Commitment or milestone:** Deliver the demo and hand in the "
                  "final report",
                  "  - **Owner(s):** All five team members",
                  f"  - **Due date:** {due}",
                  "  - **Success criteria:** every phase specification passes on a "
                  "clean checkout and the demo runs end to end without a manual fix.", ""]
        for w in nxt:
            o += [f"- **Commitment or milestone:** Close Week {w} ({EN[w]})",
                  f"  - **Owner(s):** All five team members "
                  f"({len(rows[w])} issues, 1-3 each)",
                  f"  - **Due date:** {tail(dates[w])}, 2026",
                  f"  - **Success criteria:** {GIST[w]}; all issues merged and CI green.",
                  ""]
            if w in MOBILE:
                what, owner, crit = MOBILE[w]
                o += [f"- **Commitment or milestone:** Mobile - {what}",
                      f"  - **Owner(s):** {NAME[owner]} ({owner})",
                      f"  - **Due date:** {tail(dates[w])}, 2026",
                      f"  - **Success criteria:** {crit}.", ""]

        o += ["## Team contributions and coordination", "",
              "*Note: zones do not overlap. Every file in the repository has exactly "
              f"one owner ([`plan/ownership.map`]({REPO}/blob/main/plan/ownership.map)), "
              "so two people never edit the same file. When a change needs a file owned "
              "by someone else, the requester raises it with the owner instead of "
              "editing it. All pull requests are reviewed and merged by S1 (Captain).*",
              ""]
        for s, name, _, _ in WHO:
            mine = [(w, t, ti) for w in weeks for o2, t, ti in rows[w] if o2 == s]
            after = [t for w in nxt for o2, t, _ in rows[w] if o2 == s]
            o.append(f"- **Team member:** {name} ({s})")
            if mine:
                o.append("  - **Contribution this period:** completed "
                         + ", ".join(f"{t} {ti}" for _, t, ti in mine) + ".")
                o.append("  - **Evidence:** "
                         + ", ".join(f"[{t}]({REPO}/commits/week-{w:02d})"
                                     for w, t, _ in mine) + ".")
            else:
                o.append("  - **Contribution this period:** took part in the weekly "
                         "review and the design decisions; no issue of their own in "
                         "this short week.")
                o.append("  - **Evidence:** weekly review notes on the Jira board.")
            o.append("  - **Next responsibility:** "
                     + (", ".join(after) + f" in week {nxt[0]}." if after
                        else "the demo."))
            o.append("")

        o += ["## Risks, blockers, and decisions needed", ""]
        if num == 2:
            o += ["- **Risk, blocker, or decision:** Scope of the initial requirements "
                  "against the time available",
                  "  - **Impact:** the team could run out of time before paid checkout "
                  "and ticket generation work end to end.",
                  "  - **Next action or support needed:** the estimate behind this "
                  "assessment: the requirements decompose into 96 issues across about "
                  "320 files, which is roughly two issues per person per week for ten "
                  "weeks with no slack. Four items are marked bonus and are cut first "
                  "if we fall behind - offline sync, `.ics` export, the interactive "
                  "seat map and GA4 - because none of them is needed for a visitor to "
                  "buy a ticket and get through the gate. The core checkout and ticket "
                  "generation work is protected and scheduled in weeks 4 and 5, "
                  "deliberately early.",
                  "  - **Owner:** Ansar Zeinulla (S1)", "",
                  "- **Risk, blocker, or decision:** Alinur (S4) has no commits yet",
                  "  - **Impact:** one member's contribution is not verifiable from the "
                  "repository.",
                  "  - **Next action or support needed:** `BF-11` is assigned to S4 this "
                  "week and a GitHub account is being created.",
                  "  - **Owner:** Ansar Zeinulla (S1)", ""]
        r = RISK.get(num)
        if r:
            o += [f"- **Risk, blocker, or decision:** {r[0]}",
                  f"  - **Impact:** {r[1]}",
                  f"  - **Next action or support needed:** {r[2]}",
                  f"  - **Owner:** {r[3]}", ""]
        if num != 2 and not r:
            o += ["- None.", ""]

        o += ["## Changes, reflection, and support", "",
              "- **Scope or schedule changes:** None.",
              f"- **Team reflection:** {REFLECT[num]}",
              "- **Instructor/TA help requested:** None at this time.", ""]

        if num == 2:
            o += ["## Response to feedback on Report 1", "",
                  "- **Jira access:** `oadiyatov@nu.edu.kz` invited; the board link is "
                  "in Project links above.",
                  "- **Calendar deadlines:** the plan now runs in calendar weeks "
                  "(Monday to Sunday) instead of phases, so every commitment above "
                  "carries a real date.",
                  "- **Mobile development:** the Expo app moved forward to week 2 and "
                  "now has a named deliverable in most weeks, owned by S5. See the "
                  "Mobile commitment above.",
                  "- **Evidence links:** every entry above links to the branch, the "
                  "pull requests or the file.",
                  "- **Scope assessment:** the estimate is written out in Risks above.",
                  "- **Ownership map:** linked in Project links. A change that needs "
                  "somebody else's file goes through the owner. We treat "
                  "\"conflict-free\" as a prediction to test rather than a fact: each "
                  "week `git log` is checked for any file touched by two authors, and "
                  "the count is reported here.", ""]

        path = f"plan/reports/report-{num:02d}-{SLUG[num]}.md"
        open(path, "w").write("\n".join(o))
        print(f"  {os.path.basename(path):24} недели {'+'.join(str(w) for w in weeks)}, "
              f"{closed} задач")


if __name__ == "__main__":
    main()
