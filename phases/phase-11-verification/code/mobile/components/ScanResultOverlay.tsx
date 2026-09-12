import { Pressable, StyleSheet, Text, View } from "react-native";

import { radius, theme } from "../lib/theme";
import type { CheckInStats } from "../lib/types";

export type ScanOutcome =
  | {
      kind: "valid";
      attendeeName: string;
      ticketType: string;
      seat?: string;
      stats: CheckInStats;
      /**
       * The ticket just admitted, so the admission can be undone from this
       * screen (SRS 4.8: "Undo an accidental check-in where authorized").
       *
       * The moment somebody realises they scanned the wrong person is while
       * this screen is still up, so the undo has to be here rather than on a
       * separate screen they would have to go and find.
       */
      ticketID: string;
    }
  | {
      kind: "denied";
      title: string;
      message: string;
      attendeeName?: string;
      checkedInAt?: string;
      stats?: CheckInStats;
    };

/**
 * The whole point of the scanner: a result readable at a glance, from arm's
 * length, by someone who is not looking closely.
 *
 * It fills the screen so the colour alone answers "let them in?" before any
 * text is read - but the colour is never the only signal. Each state also
 * carries a large symbol and a heading, because roughly one man in twelve
 * cannot reliably tell the green from the red (SRS 4.8, WCAG 2.1 AA).
 */
export function ScanResultOverlay({
  outcome,
  onDismiss,
  onUndo,
  undoing = false,
}: {
  outcome: ScanOutcome;
  onDismiss: () => void;
  /** Reverse the admission this screen is reporting (SRS 4.8). */
  onUndo?: (ticketID: string) => void;
  undoing?: boolean;
}) {
  const valid = outcome.kind === "valid";
  const stats = outcome.stats;
  const canUndo = valid && onUndo !== undefined;

  return (
    <Pressable
      style={[styles.fill, { backgroundColor: valid ? theme.successBright : theme.dangerBright }]}
      onPress={onDismiss}
      accessibilityRole="button"
      accessibilityLabel={valid ? "Valid admission" : "Admission denied"}
      accessibilityLiveRegion="assertive"
      testID={valid ? "result-valid" : "result-denied"}
    >
      <View style={styles.content}>
        <Text style={styles.symbol}>{valid ? "✓" : "✕"}</Text>

        <Text style={styles.heading} testID="result-heading">
          {valid ? "CHECKED IN" : outcome.title.toUpperCase()}
        </Text>

        {valid ? (
          <>
            <Text style={styles.name} testID="result-name">
              {outcome.attendeeName}
            </Text>
            <Text style={styles.detail}>{outcome.ticketType}</Text>
            {outcome.seat ? <Text style={styles.detail}>{outcome.seat}</Text> : null}
          </>
        ) : (
          <>
            {outcome.attendeeName ? (
              <Text style={styles.name} testID="result-name">
                {outcome.attendeeName}
              </Text>
            ) : null}
            <Text style={styles.message}>{outcome.message}</Text>
            {outcome.checkedInAt ? (
              <Text style={styles.detail}>
                First used at{" "}
                {new Date(outcome.checkedInAt).toLocaleTimeString([], {
                  hour: "2-digit",
                  minute: "2-digit",
                })}
              </Text>
            ) : null}
          </>
        )}

        {stats ? (
          <View style={styles.statsRow}>
            <Text style={styles.stats}>
              {stats.checked_in} of {stats.issued} checked in
            </Text>
          </View>
        ) : null}
      </View>

      {canUndo ? (
        <Pressable
          // The undo sits inside the full-screen Pressable, so its own press
          // must not also count as "dismiss and scan the next person".
          onPress={(event) => {
            event.stopPropagation();
            if (!undoing) onUndo(outcome.ticketID);
          }}
          disabled={undoing}
          style={styles.undo}
          accessibilityRole="button"
          accessibilityLabel="Undo this check-in"
          testID="undo-check-in"
        >
          <Text style={styles.undoText}>
            {undoing ? "Undoing…" : "Wrong person? Undo"}
          </Text>
        </Pressable>
      ) : null}

      <Text style={styles.dismiss}>
        {valid ? "Tap anywhere to scan the next ticket" : "Tap when you have dealt with this"}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  fill: {
    position: "absolute",
    top: 0,
    right: 0,
    bottom: 0,
    left: 0,
    alignItems: "center",
    justifyContent: "center",
    padding: 28,
    zIndex: 10,
  },
  content: { alignItems: "center", gap: 6 },
  undo: {
    marginTop: 20,
    borderWidth: 2,
    borderColor: "rgba(255,255,255,0.85)",
    borderRadius: 999,
    paddingHorizontal: 22,
    paddingVertical: 11,
  },
  undoText: { color: "#fff", fontSize: 16, fontWeight: "700" },
  symbol: { fontSize: 96, color: "#fff", fontWeight: "300", lineHeight: 104 },
  heading: {
    fontSize: 30,
    fontWeight: "800",
    color: "#fff",
    letterSpacing: 1,
    textAlign: "center",
    marginTop: 4,
  },
  name: {
    fontSize: 26,
    fontWeight: "700",
    color: "#fff",
    textAlign: "center",
    marginTop: 10,
  },
  detail: { fontSize: 16, color: "rgba(255,255,255,0.9)", textAlign: "center" },
  message: {
    fontSize: 17,
    color: "#fff",
    textAlign: "center",
    marginTop: 8,
    paddingHorizontal: 12,
  },
  statsRow: {
    marginTop: 26,
    backgroundColor: "rgba(0,0,0,0.22)",
    borderRadius: radius.md,
    paddingHorizontal: 16,
    paddingVertical: 8,
  },
  stats: { color: "#fff", fontSize: 14, fontWeight: "600" },
  dismiss: {
    position: "absolute",
    bottom: 44,
    color: "rgba(255,255,255,0.85)",
    fontSize: 14,
  },
});
