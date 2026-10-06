import { Stack, useLocalSearchParams } from "expo-router";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  Vibration,
  View,
} from "react-native";

import { ApiError, api } from "../../lib/api";
import { radius, theme } from "../../lib/theme";
import type { AttendeeTicket } from "../../lib/types";

/**
 * How long to wait after a keystroke before searching.
 *
 * Long enough that typing a name is one request rather than eight, short
 * enough that it still feels like the list is following along.
 */
const DEBOUNCE_MS = 300;

/** How long a confirmation stays on the row after a manual admission. */
const CONFIRMATION_MS = 4000;

const STATUS_LABELS: Record<AttendeeTicket["status"], string> = {
  valid: "Not yet in",
  checked_in: "Already in",
  cancelled: "Cancelled",
  refunded: "Refunded",
};

/**
 * Manual attendee search (SRS 4.8, "search for attendees manually").
 *
 * The screen staff open when a QR will not scan: a cracked phone screen, a dead
 * battery, a printout left at home. Finding somebody by name and admitting them
 * is a real check-in - the server runs the same transaction as a scan - so it
 * is subject to the same duplicate protection rather than being a way around
 * it.
 */
export default function AttendeeSearchScreen() {
  const { eventId } = useLocalSearchParams<{ eventId: string }>();

  const [query, setQuery] = useState("");
  const [attendees, setAttendees] = useState<AttendeeTicket[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Which ticket is mid-request, and the last thing that happened to a row.
  const [admitting, setAdmitting] = useState<string | null>(null);
  const [confirmation, setConfirmation] = useState<{ id: string; text: string } | null>(null);

  // Guards against a slow early response overwriting a newer one.
  const requestSeq = useRef(0);

  const search = useCallback(
    async (term: string) => {
      const seq = ++requestSeq.current;
      try {
        const found = await api.searchAttendees(eventId, term);
        if (seq !== requestSeq.current) return;
        setAttendees(found);
        setError(null);
      } catch (cause) {
        if (seq !== requestSeq.current) return;
        setError(
          cause instanceof ApiError ? cause.message : "Could not reach the server.",
        );
      } finally {
        if (seq === requestSeq.current) setLoading(false);
      }
    },
    [eventId],
  );

  useEffect(() => {
    const timer = setTimeout(() => void search(query.trim()), query === "" ? 0 : DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [query, search]);

  useEffect(() => {
    if (!confirmation) return;
    const timer = setTimeout(() => setConfirmation(null), CONFIRMATION_MS);
    return () => clearTimeout(timer);
  }, [confirmation]);

  /**
   * Reverse an admission from the list (SRS 4.8: "Undo an accidental check-in
   * where authorized").
   *
   * The scanner's own overlay covers the case where the mistake is noticed
   * immediately; this covers the one where it is noticed a minute later, which
   * is when somebody comes back to the desk to say so.
   */
  async function undo(attendee: AttendeeTicket) {
    setAdmitting(attendee.ticket_id);
    try {
      await api.reverseCheckIn(attendee.ticket_id, "reversed from attendee search");
      if (Platform.OS !== "web") Vibration.vibrate(40);
      setConfirmation({
        id: attendee.ticket_id,
        text: `Check-in undone: ${attendee.attendee_name}`,
      });
    } catch (cause) {
      setConfirmation({
        id: attendee.ticket_id,
        text:
          cause instanceof ApiError
            ? cause.message
            : "Could not undo this check-in. Try again.",
      });
    } finally {
      setAdmitting(null);
      await search(query.trim());
    }
  }

  async function admit(attendee: AttendeeTicket) {
    setAdmitting(attendee.ticket_id);
    try {
      const result = await api.checkInManually(eventId, attendee.ticket_id, "manual search");

      // The same short buzz the scanner gives: staff are looking at the
      // attendee, not at the screen.
      if (Platform.OS !== "web") Vibration.vibrate(40);

      setConfirmation({
        id: attendee.ticket_id,
        text: `Checked in: ${result.attendee_name}`,
      });
      await search(query.trim());
    } catch (cause) {
      setConfirmation({
        id: attendee.ticket_id,
        text:
          cause instanceof ApiError
            ? cause.message
            : "Could not check this ticket in. Try again.",
      });
      // Refresh anyway: the refusal usually means the row is out of date.
      await search(query.trim());
    } finally {
      setAdmitting(null);
    }
  }

  return (
    <View style={styles.screen}>
      <Stack.Screen options={{ title: "Find attendee" }} />

      <View style={styles.searchBar}>
        <TextInput
          value={query}
          onChangeText={setQuery}
          placeholder="Name, email, ticket or order number"
          placeholderTextColor={theme.textMuted}
          autoCapitalize="none"
          autoCorrect={false}
          returnKeyType="search"
          style={styles.input}
          testID="attendee-search-input"
        />
      </View>

      {error ? (
        <View style={styles.banner}>
          <Text style={styles.bannerText}>{error}</Text>
        </View>
      ) : null}

      {loading ? (
        <View style={styles.centered}>
          <ActivityIndicator color={theme.brand} />
        </View>
      ) : (
        <FlatList
          data={attendees}
          keyExtractor={(item) => item.ticket_id}
          contentContainerStyle={styles.list}
          keyboardShouldPersistTaps="handled"
          ListEmptyComponent={
            <View style={styles.centered}>
              <Text style={styles.emptyTitle}>
                {query.trim() === "" ? "No tickets yet" : "Nobody matches that"}
              </Text>
              <Text style={styles.muted}>
                {query.trim() === ""
                  ? "Nobody has bought a ticket for this event."
                  : "Try a surname, an email address, or the order number."}
              </Text>
            </View>
          }
          renderItem={({ item }) => {
            const busy = admitting === item.ticket_id;
            const note = confirmation?.id === item.ticket_id ? confirmation.text : null;

            return (
              <View style={styles.card} testID={`attendee-${item.ticket_id}`}>
                <View style={styles.cardHeader}>
                  <Text style={styles.name} numberOfLines={1}>
                    {item.attendee_name}
                  </Text>
                  <View
                    style={[
                      styles.badge,
                      item.status === "valid" ? styles.badgeValid : styles.badgeSpent,
                    ]}
                  >
                    <Text style={styles.badgeText}>{STATUS_LABELS[item.status]}</Text>
                  </View>
                </View>

                <Text style={styles.muted} numberOfLines={1}>
                  {item.attendee_email}
                </Text>
                <Text style={styles.muted}>
                  {item.ticket_type_name} · {item.ticket_code}
                </Text>

                {note ? (
                  <View style={styles.note}>
                    <Text style={styles.noteText}>{note}</Text>
                  </View>
                ) : null}

                {item.status === "checked_in" ? (
                  <Pressable
                    disabled={busy}
                    onPress={() => void undo(item)}
                    testID={`undo-${item.ticket_id}`}
                    style={({ pressed }) => [
                      styles.button,
                      styles.undoButton,
                      pressed && styles.buttonPressed,
                    ]}
                  >
                    {busy ? (
                      <ActivityIndicator color={theme.text} />
                    ) : (
                      <Text style={[styles.buttonText, styles.undoButtonText]}>
                        Undo check-in
                      </Text>
                    )}
                  </Pressable>
                ) : (
                  <Pressable
                    disabled={!item.admissible || busy}
                    onPress={() => void admit(item)}
                    testID={`admit-${item.ticket_id}`}
                    style={({ pressed }) => [
                      styles.button,
                      !item.admissible && styles.buttonDisabled,
                      pressed && item.admissible && styles.buttonPressed,
                    ]}
                  >
                    {busy ? (
                      <ActivityIndicator color="#fff" />
                    ) : (
                      <Text style={styles.buttonText}>
                        {item.admissible ? "Check in" : `Ticket ${item.status}`}
                      </Text>
                    )}
                  </Pressable>
                )}
              </View>
            );
          }}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: theme.bg },
  searchBar: { padding: 16, paddingBottom: 8 },
  input: {
    backgroundColor: theme.surface,
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: radius.md,
    color: theme.text,
    fontSize: 16,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  banner: {
    backgroundColor: theme.danger,
    marginHorizontal: 16,
    marginBottom: 8,
    padding: 12,
    borderRadius: radius.sm,
  },
  bannerText: { color: "#fff", fontSize: 14 },
  centered: { alignItems: "center", justifyContent: "center", padding: 32, gap: 6 },
  emptyTitle: { color: theme.text, fontSize: 16, fontWeight: "600" },
  muted: { color: theme.textMuted, fontSize: 14, textAlign: "center" },
  list: { padding: 16, paddingTop: 8, gap: 12 },
  card: {
    backgroundColor: theme.surface,
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: radius.md,
    padding: 14,
    gap: 4,
  },
  cardHeader: { flexDirection: "row", alignItems: "center", gap: 8 },
  name: { color: theme.text, fontSize: 17, fontWeight: "600", flexShrink: 1 },
  badge: { borderRadius: 999, paddingHorizontal: 10, paddingVertical: 3, marginLeft: "auto" },
  badgeValid: { backgroundColor: theme.brandDark },
  badgeSpent: { backgroundColor: theme.surfaceAlt },
  badgeText: { color: theme.text, fontSize: 12, fontWeight: "600" },
  note: {
    backgroundColor: theme.surfaceAlt,
    borderRadius: radius.sm,
    padding: 10,
    marginTop: 8,
  },
  noteText: { color: theme.text, fontSize: 14 },
  button: {
    marginTop: 10,
    backgroundColor: theme.success,
    borderRadius: radius.sm,
    paddingVertical: 14,
    alignItems: "center",
    justifyContent: "center",
  },
  buttonPressed: { opacity: 0.85 },
  buttonDisabled: { backgroundColor: theme.surfaceAlt },
  // Outlined rather than filled: undoing is a correction, not the action the
  // person at the desk is normally reaching for.
  undoButton: {
    backgroundColor: "transparent",
    borderWidth: 2,
    borderColor: theme.border,
  },
  undoButtonText: { color: theme.text },
  buttonText: { color: "#fff", fontSize: 16, fontWeight: "700" },
});
