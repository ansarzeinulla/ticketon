import { CameraView, useCameraPermissions } from "expo-camera";
import { Stack, router, useLocalSearchParams } from "expo-router";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  Vibration,
  View,
} from "react-native";

import { ScanResultOverlay, type ScanOutcome } from "../../components/ScanResultOverlay";
import { ApiError, api } from "../../lib/api";
import { useAuth } from "../../lib/auth-context";
import { radius, theme } from "../../lib/theme";
import type { CheckInStats } from "../../lib/types";

/**
 * How long a green result fills the screen before the camera resumes.
 *
 * Only the green one auto-dismisses. A queue has to keep moving, and a valid
 * admission needs no decision from staff. A refusal does: someone has to read
 * why, talk to the attendee and decide what happens next, so the red screen
 * stays up until it is dismissed deliberately.
 */
const VALID_RESULT_DISPLAY_MS = 2600;

/**
 * Ignore repeat frames of the same code for this long. A camera decodes many
 * frames a second, so without this one physical ticket would fire dozens of
 * check-in requests and every one after the first would come back "already
 * used" - the app would accuse the attendee of double entry it caused itself.
 */
const SAME_CODE_COOLDOWN_MS = 3000;

/** Turns an API failure into the words on the red screen. */
function outcomeFromError(error: unknown): ScanOutcome {
  if (!(error instanceof ApiError)) {
    return { kind: "denied", title: "Error", message: "Something went wrong. Try again." };
  }

  switch (error.code) {
    case "already_checked_in":
      return {
        kind: "denied",
        title: "Already used",
        message: "This ticket has already been scanned at the entrance.",
        attendeeName: error.attendeeName,
        checkedInAt: error.checkedInAt,
        stats: error.stats,
      };
    case "campaign_token":
      // SRS 4.14: a campaign QR must never open a gate. Naming it precisely
      // matters - staff can then tell the attendee they scanned the poster
      // rather than their ticket, instead of turning away a valid holder.
      return {
        kind: "denied",
        title: "Promo code, not a ticket",
        message:
          "This is a promotional discount code from an advert or poster. " +
          "Ask the attendee for the QR code on their ticket or PDF.",
      };
    case "wrong_event":
      return { kind: "denied", title: "Wrong event", message: error.message };
    case "ticket_not_valid":
      return {
        kind: "denied",
        title: "Invalid ticket",
        message: error.message,
        attendeeName: error.attendeeName,
      };
    case "unknown_ticket":
      return {
        kind: "denied",
        title: "Invalid",
        message: "This code does not match any ticket for this event.",
      };
    case "forbidden":
      return { kind: "denied", title: "Not authorised", message: error.message };
    case "network_error":
      return { kind: "denied", title: "Offline", message: error.message };
    default:
      return { kind: "denied", title: "Denied", message: error.message };
  }
}

export default function ScannerScreen() {
  const { eventId } = useLocalSearchParams<{ eventId: string }>();
  const { user, signOut } = useAuth();

  const [permission, requestPermission] = useCameraPermissions();
  const [outcome, setOutcome] = useState<ScanOutcome | null>(null);
  const [undoing, setUndoing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [stats, setStats] = useState<CheckInStats | null>(null);
  const [manualOpen, setManualOpen] = useState(false);
  const [manualCode, setManualCode] = useState("");

  // Refs, not state: the camera callback fires on frames between renders, so
  // these have to be readable and writable without waiting for a re-render.
  const lastCode = useRef<{ value: string; at: number } | null>(null);
  const inFlight = useRef(false);
  const dismissTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const deviceLabel = `${Platform.OS} scanner`;

  useEffect(() => {
    return () => {
      if (dismissTimer.current) clearTimeout(dismissTimer.current);
    };
  }, []);

  const refreshStats = useCallback(async () => {
    try {
      setStats(await api.stats(eventId));
    } catch {
      // The counter is informational; a failure here must not break scanning.
    }
  }, [eventId]);

  useEffect(() => {
    void refreshStats();
  }, [refreshStats]);

  const showOutcome = useCallback((next: ScanOutcome) => {
    setOutcome(next);
    // A short buzz for success, a longer double buzz for refusal: staff often
    // work with the screen at their side rather than in front of them.
    Vibration.vibrate(next.kind === "valid" ? 40 : [0, 120, 90, 120]);

    if (dismissTimer.current) clearTimeout(dismissTimer.current);
    if (next.kind === "valid") {
      dismissTimer.current = setTimeout(() => setOutcome(null), VALID_RESULT_DISPLAY_MS);
    }
  }, []);

  /**
   * Reverse the admission on screen (SRS 4.8: "Undo an accidental check-in
   * where authorized").
   *
   * The auto-dismiss timer is cancelled first: somebody reaching for Undo is
   * mid-decision, and having the screen vanish under their thumb is how the
   * wrong person ends up admitted anyway.
   */
  const undoCheckIn = useCallback(
    async (ticketID: string) => {
      if (dismissTimer.current) clearTimeout(dismissTimer.current);
      setUndoing(true);
      try {
        const updated = await api.reverseCheckIn(ticketID, "reversed at the gate");
        setStats(updated);
        setOutcome(null);
        // A single buzz confirms the reversal landed, so staff know without
        // reading the screen that the person is off the list again.
        Vibration.vibrate(40);
      } catch (error) {
        if (error instanceof ApiError && error.isSessionExpired) {
          await signOut();
          router.replace("/login");
          return;
        }
        showOutcome({
          kind: "denied",
          title: "Undo failed",
          message:
            error instanceof ApiError
              ? error.message
              : "The check-in could not be reversed. Try again.",
        });
      } finally {
        setUndoing(false);
      }
    },
    [showOutcome, signOut],
  );

  const submit = useCallback(
    async (token: string) => {
      if (inFlight.current) return;
      inFlight.current = true;
      setBusy(true);

      try {
        const result = await api.checkIn(eventId, token, deviceLabel);
        setStats(result.stats);
        showOutcome({
          kind: "valid",
          attendeeName: result.attendee_name,
          ticketType: result.ticket_type_name,
          seat: result.seat_label || undefined,
          stats: result.stats,
          ticketID: result.ticket_id,
        });
      } catch (error) {
        if (error instanceof ApiError && error.isSessionExpired) {
          await signOut();
          router.replace("/login");
          return;
        }
        if (error instanceof ApiError && error.stats) setStats(error.stats);
        showOutcome(outcomeFromError(error));
      } finally {
        inFlight.current = false;
        setBusy(false);
      }
    },
    [eventId, deviceLabel, showOutcome, signOut],
  );

  const handleBarcode = useCallback(
    ({ data }: { data: string }) => {
      const code = data.trim();
      if (!code) return;

      const now = Date.now();
      const previous = lastCode.current;
      if (previous && previous.value === code && now - previous.at < SAME_CODE_COOLDOWN_MS) {
        return;
      }
      lastCode.current = { value: code, at: now };

      void submit(code);
    },
    [submit],
  );

  function handleManualSubmit() {
    const code = manualCode.trim();
    if (!code) return;
    setManualCode("");
    setManualOpen(false);
    void submit(code);
  }

  // --- camera permission states --------------------------------------------
  if (!permission) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator color={theme.brand} size="large" />
      </View>
    );
  }

  if (!permission.granted) {
    return (
      <View style={styles.centered}>
        <Text style={styles.permissionTitle}>Camera access needed</Text>
        <Text style={styles.muted}>
          BiletFlow uses the camera to read ticket QR codes at the entrance.
        </Text>
        <Pressable style={styles.primaryButton} onPress={requestPermission}>
          <Text style={styles.primaryButtonText}>Allow camera</Text>
        </Pressable>
        <Pressable onPress={() => setManualOpen(true)}>
          <Text style={styles.link}>Enter a code by hand instead</Text>
        </Pressable>
        {manualOpen && (
          <ManualEntry
            value={manualCode}
            onChange={setManualCode}
            onSubmit={handleManualSubmit}
            onCancel={() => setManualOpen(false)}
            busy={busy}
          />
        )}
        {outcome && (
          <ScanResultOverlay
            outcome={outcome}
            onDismiss={() => setOutcome(null)}
            onUndo={undoCheckIn}
            undoing={undoing}
          />
        )}
      </View>
    );
  }

  return (
    <View style={styles.flex}>
      <Stack.Screen options={{ title: "Scan tickets" }} />

      <CameraView
        style={StyleSheet.absoluteFill}
        facing="back"
        barcodeScannerSettings={{ barcodeTypes: ["qr"] }}
        // Frames stop being handled while a result is up or a request is in
        // flight, so the next attendee's code cannot be swallowed by the last
        // one's result.
        onBarcodeScanned={outcome || busy ? undefined : handleBarcode}
      />

      <View style={styles.overlay} pointerEvents="box-none">
        <View style={styles.reticle} />
        <Text style={styles.hint}>Point the camera at the ticket QR code</Text>
      </View>

      <View style={styles.bottomBar} pointerEvents="box-none">
        {stats && (
          <Text style={styles.counter} testID="scan-counter">
            {stats.checked_in} of {stats.issued} checked in
          </Text>
        )}
        <Pressable
          style={styles.manualButton}
          onPress={() => setManualOpen(true)}
          testID="open-manual-entry"
        >
          <Text style={styles.manualButtonText}>Enter code by hand</Text>
        </Pressable>
        {/*
          The last resort when there is no code to enter at all: a dead phone,
          a ticket left at home (SRS 4.8).
        */}
        <Pressable
          style={styles.manualButton}
          onPress={() => router.push(`/attendees/${eventId}`)}
          testID="open-attendee-search"
        >
          <Text style={styles.manualButtonText}>Find attendee by name</Text>
        </Pressable>
        <Text style={styles.who}>{user?.email}</Text>
      </View>

      {busy && (
        <View style={styles.busy} pointerEvents="none">
          <ActivityIndicator color="#fff" size="large" />
        </View>
      )}

      {manualOpen && (
        <ManualEntry
          value={manualCode}
          onChange={setManualCode}
          onSubmit={handleManualSubmit}
          onCancel={() => setManualOpen(false)}
          busy={busy}
        />
      )}

      {outcome && (
          <ScanResultOverlay
            outcome={outcome}
            onDismiss={() => setOutcome(null)}
            onUndo={undoCheckIn}
            undoing={undoing}
          />
        )}
    </View>
  );
}

/**
 * Typing a code by hand.
 *
 * Not a debug affordance: a scratched ticket or a dead camera still has to let
 * someone in, and SRS 4.8 requires staff to be able to find an attendee without
 * scanning. It is also the only way to exercise the gate on a simulator, which
 * has no camera.
 */
function ManualEntry({
  value,
  onChange,
  onSubmit,
  onCancel,
  busy,
}: {
  value: string;
  onChange: (next: string) => void;
  onSubmit: () => void;
  onCancel: () => void;
  busy: boolean;
}) {
  return (
    <View style={styles.sheet}>
      <Text style={styles.sheetTitle}>Enter the ticket code</Text>
      <Text style={styles.muted}>
        The code printed under the QR on the ticket, starting with TKT_.
      </Text>

      <TextInput
        style={styles.sheetInput}
        value={value}
        onChangeText={onChange}
        placeholder="TKT_00000000-0000-0000-0000-000000000000"
        placeholderTextColor={theme.textMuted}
        autoCapitalize="none"
        autoCorrect={false}
        autoFocus
        editable={!busy}
        onSubmitEditing={onSubmit}
        returnKeyType="go"
        testID="manual-code-input"
      />

      <View style={styles.sheetActions}>
        <Pressable
          style={[styles.primaryButton, styles.primaryButtonInRow]}
          onPress={onSubmit}
          disabled={busy}
          testID="manual-submit"
        >
          <Text style={styles.primaryButtonText}>Check in</Text>
        </Pressable>
        <Pressable style={styles.secondaryButton} onPress={onCancel} disabled={busy}>
          <Text style={styles.secondaryButtonText}>Cancel</Text>
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: "#000" },
  centered: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 14,
    padding: 32,
    backgroundColor: theme.bg,
  },
  muted: { color: theme.textMuted, fontSize: 14, textAlign: "center" },
  permissionTitle: { color: theme.text, fontSize: 20, fontWeight: "700" },
  link: { color: theme.brand, fontSize: 15, fontWeight: "600", marginTop: 8 },
  overlay: { flex: 1, alignItems: "center", justifyContent: "center", gap: 22 },
  reticle: {
    width: 250,
    height: 250,
    borderRadius: radius.lg,
    borderWidth: 3,
    borderColor: "rgba(255,255,255,0.9)",
  },
  hint: { color: "#fff", fontSize: 15, textShadowColor: "#000", textShadowRadius: 6 },
  bottomBar: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    alignItems: "center",
    gap: 10,
    paddingBottom: 34,
    paddingTop: 18,
    backgroundColor: "rgba(0,0,0,0.55)",
  },
  counter: { color: "#fff", fontSize: 16, fontWeight: "700" },
  manualButton: {
    borderColor: "rgba(255,255,255,0.6)",
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: 18,
    paddingVertical: 10,
  },
  manualButtonText: { color: "#fff", fontSize: 14, fontWeight: "600" },
  who: { color: "rgba(255,255,255,0.6)", fontSize: 12 },
  busy: {
    position: "absolute",
    top: 0,
    right: 0,
    bottom: 0,
    left: 0,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: "rgba(0,0,0,0.35)",
  },
  sheet: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: theme.surface,
    borderTopLeftRadius: 22,
    borderTopRightRadius: 22,
    padding: 24,
    paddingBottom: 40,
    gap: 10,
    zIndex: 5,
  },
  sheetTitle: { color: theme.text, fontSize: 19, fontWeight: "700" },
  sheetInput: {
    backgroundColor: theme.surfaceAlt,
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: 14,
    paddingVertical: 14,
    color: theme.text,
    fontSize: 15,
    marginTop: 6,
  },
  sheetActions: { flexDirection: "row", gap: 12, marginTop: 8 },
  primaryButton: {
    backgroundColor: theme.brandDark,
    borderRadius: radius.md,
    paddingVertical: 15,
    paddingHorizontal: 28,
    alignItems: "center",
    // Deliberately not flex:1 - this button is used both on its own and inside
    // the sheet's action row, and stretching is only wanted in the row.
    alignSelf: "center",
  },
  primaryButtonInRow: { flex: 1, alignSelf: "auto" },
  primaryButtonText: { color: "#fff", fontSize: 16, fontWeight: "600" },
  secondaryButton: {
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingVertical: 15,
    paddingHorizontal: 22,
    alignItems: "center",
  },
  secondaryButtonText: { color: theme.text, fontSize: 16, fontWeight: "600" },
});
