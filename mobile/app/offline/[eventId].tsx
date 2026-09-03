import { CameraView, useCameraPermissions } from "expo-camera";
import { Stack, router, useLocalSearchParams } from "expo-router";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  Vibration,
  View,
} from "react-native";

import { ScanResultOverlay, type ScanOutcome } from "../../components/ScanResultOverlay";
import { ApiError, api } from "../../lib/api";
import { useAuth } from "../../lib/auth-context";
import { pendingCount, rosterStats, saveRoster, undoLocalCheckIn } from "../../lib/offline-db";
import { syncPending, verifyOffline } from "../../lib/offline";
import { radius, theme } from "../../lib/theme";
import { useConnectivity } from "../../lib/use-connectivity";

const VALID_RESULT_DISPLAY_MS = 2600;
const SAME_CODE_COOLDOWN_MS = 3000;

/**
 * The offline gate (SRS 4.8).
 *
 * Same door, no network. It validates against a roster downloaded in advance -
 * comparing the hash of each scanned token, never a plaintext credential - and
 * queues every admission to reconcile with the server the moment a connection
 * comes back. The screen never blocks on the network: a queue at a venue
 * entrance cannot wait for Wi-Fi.
 */
export default function OfflineScannerScreen() {
  const { eventId } = useLocalSearchParams<{ eventId: string }>();
  const { user, signOut } = useAuth();
  const { online } = useConnectivity();

  const [permission, requestPermission] = useCameraPermissions();
  const [outcome, setOutcome] = useState<ScanOutcome | null>(null);
  const [loadingRoster, setLoadingRoster] = useState(true);
  const [rosterError, setRosterError] = useState<string | null>(null);
  const [stats, setStats] = useState({ total: 0, checkedIn: 0, hasRoster: false });
  const [pending, setPending] = useState(0);
  const [syncing, setSyncing] = useState(false);
  const [syncNote, setSyncNote] = useState<string | null>(null);
  const [manualOpen, setManualOpen] = useState(false);
  const [manualCode, setManualCode] = useState("");

  const lastCode = useRef<{ value: string; at: number } | null>(null);
  const inFlight = useRef(false);
  const dismissTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const deviceLabel = `${Platform.OS} scanner (offline)`;

  useEffect(() => {
    return () => {
      if (dismissTimer.current) clearTimeout(dismissTimer.current);
    };
  }, []);

  const refreshCounts = useCallback(async () => {
    const [s, p] = await Promise.all([rosterStats(eventId), pendingCount(eventId)]);
    setStats(s);
    setPending(p);
  }, [eventId]);

  /** Pull a fresh roster while there is a network to pull it with. */
  const downloadRoster = useCallback(async () => {
    setRosterError(null);
    try {
      const roster = await api.roster(eventId);
      await saveRoster(roster);
    } catch (cause) {
      if (cause instanceof ApiError && cause.isSessionExpired) {
        await signOut();
        router.replace("/login");
        return;
      }
      // A failure here is not fatal: an earlier roster may already be on the
      // device, which is the whole point of having downloaded it.
      setRosterError(
        cause instanceof ApiError && cause.isNetworkError
          ? "Offline - using the roster already on this device."
          : cause instanceof ApiError
            ? cause.message
            : "Could not download the roster.",
      );
    } finally {
      await refreshCounts();
      setLoadingRoster(false);
    }
  }, [eventId, refreshCounts, signOut]);

  useEffect(() => {
    void downloadRoster();
  }, [downloadRoster]);

  const runSync = useCallback(
    async (silent: boolean) => {
      if (syncing) return;
      setSyncing(true);
      if (!silent) setSyncNote(null);
      try {
        const summary = await syncPending(eventId, deviceLabel);
        if (summary) {
          setSyncNote(
            `Synced ${summary.recorded} admission${summary.recorded === 1 ? "" : "s"}` +
              (summary.rejected > 0 ? `, ${summary.rejected} refused by the server` : "") +
              ".",
          );
        } else if (!silent) {
          setSyncNote("Nothing to sync.");
        }
      } catch (cause) {
        if (cause instanceof ApiError && cause.isSessionExpired) {
          await signOut();
          router.replace("/login");
          return;
        }
        if (!silent) {
          setSyncNote(
            cause instanceof ApiError && cause.isNetworkError
              ? "Still offline. The queue is safe and will sync when you reconnect."
              : "Sync failed. The queue is kept; try again.",
          );
        }
      } finally {
        await refreshCounts();
        setSyncing(false);
      }
    },
    [eventId, deviceLabel, syncing, refreshCounts, signOut],
  );

  // The moment the network comes back, drain the queue without being asked.
  useEffect(() => {
    if (online && pending > 0 && !syncing) {
      void runSync(true);
    }
    // runSync is intentionally omitted: this fires on the online/pending edge,
    // not on every identity change of the callback.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [online, pending]);

  const showOutcome = useCallback((next: ScanOutcome) => {
    setOutcome(next);
    Vibration.vibrate(next.kind === "valid" ? 40 : [0, 120, 90, 120]);
    if (dismissTimer.current) clearTimeout(dismissTimer.current);
    if (next.kind === "valid") {
      dismissTimer.current = setTimeout(() => setOutcome(null), VALID_RESULT_DISPLAY_MS);
    }
  }, []);

  const undo = useCallback(
    async (ticketID: string) => {
      if (dismissTimer.current) clearTimeout(dismissTimer.current);
      await undoLocalCheckIn(eventId, ticketID);
      await refreshCounts();
      setOutcome(null);
      Vibration.vibrate(40);
    },
    [eventId, refreshCounts],
  );

  const submit = useCallback(
    async (token: string) => {
      if (inFlight.current) return;
      inFlight.current = true;
      try {
        const localStats = await rosterStats(eventId);
        const result = await verifyOffline(eventId, token, deviceLabel);
        await refreshCounts();

        const counter = {
          issued: localStats.total,
          checked_in: localStats.checkedIn + (result.kind === "valid" ? 1 : 0),
        };

        switch (result.kind) {
          case "valid":
            showOutcome({
              kind: "valid",
              attendeeName: result.entry.attendee_name,
              ticketType: result.entry.ticket_type_name,
              seat: result.entry.seat_label || undefined,
              stats: counter,
              ticketID: result.entry.ticket_id,
            });
            break;
          case "already_checked_in":
            showOutcome({
              kind: "denied",
              title: "Already used",
              message: "This ticket has already been scanned at this door.",
              attendeeName: result.entry.attendee_name,
              stats: counter,
            });
            break;
          case "not_valid":
            showOutcome({
              kind: "denied",
              title: result.reason === "refunded" ? "Refunded" : "Cancelled",
              message: `This ticket is ${result.reason} and is not valid for entry.`,
              attendeeName: result.entry.attendee_name,
              stats: counter,
            });
            break;
          default:
            showOutcome({
              kind: "denied",
              title: "Invalid",
              message:
                "This code does not match any ticket in the downloaded roster " +
                "for this event.",
              stats: counter,
            });
        }
      } finally {
        inFlight.current = false;
      }
    },
    [eventId, deviceLabel, showOutcome, refreshCounts],
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

  if (!permission) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator color={theme.brand} size="large" />
      </View>
    );
  }

  // Without a roster the offline gate has nothing to check against; say so
  // rather than showing a camera that would refuse every ticket.
  if (!loadingRoster && !stats.hasRoster) {
    return (
      <View style={styles.centered}>
        <Stack.Screen options={{ title: "Offline check-in" }} />
        <Text style={styles.permissionTitle}>No roster on this device</Text>
        <Text style={styles.muted}>
          Connect to the internet once to download the guest list, then this door
          works with no network at all.
        </Text>
        {rosterError && <Text style={styles.muted}>{rosterError}</Text>}
        <Pressable style={styles.primaryButton} onPress={() => void downloadRoster()}>
          <Text style={styles.primaryButtonText}>Download roster</Text>
        </Pressable>
      </View>
    );
  }

  if (!permission.granted) {
    return (
      <View style={styles.centered}>
        <Stack.Screen options={{ title: "Offline check-in" }} />
        <Text style={styles.permissionTitle}>Camera access needed</Text>
        <Text style={styles.muted}>
          The offline gate reads the same ticket QR codes; it just checks them
          against the roster on this device.
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
          />
        )}
      </View>
    );
  }

  return (
    <View style={styles.flex}>
      <Stack.Screen options={{ title: "Offline check-in" }} />

      <CameraView
        style={StyleSheet.absoluteFill}
        facing="back"
        barcodeScannerSettings={{ barcodeTypes: ["qr"] }}
        onBarcodeScanned={outcome ? undefined : handleBarcode}
      />

      <View style={styles.topBar} pointerEvents="box-none">
        <View style={[styles.pill, online ? styles.pillOnline : styles.pillOffline]}>
          <Text style={styles.pillText}>
            {online ? "● Online" : "● Offline"}
          </Text>
        </View>
        {pending > 0 && (
          <View style={[styles.pill, styles.pillPending]}>
            <Text style={styles.pillText}>
              {pending} to sync{syncing ? "…" : ""}
            </Text>
          </View>
        )}
      </View>

      <View style={styles.overlay} pointerEvents="box-none">
        <View style={styles.reticle} />
        <Text style={styles.hint}>Working offline · checking against the roster</Text>
      </View>

      <View style={styles.bottomBar} pointerEvents="box-none">
        <Text style={styles.counter} testID="offline-counter">
          {stats.checkedIn} of {stats.total} checked in
        </Text>
        {syncNote && <Text style={styles.syncNote}>{syncNote}</Text>}

        <View style={styles.buttonRow}>
          <Pressable style={styles.manualButton} onPress={() => setManualOpen(true)}>
            <Text style={styles.manualButtonText}>Enter code by hand</Text>
          </Pressable>
          <Pressable
            style={[styles.manualButton, pending === 0 && styles.disabled]}
            onPress={() => void runSync(false)}
            disabled={pending === 0 || syncing}
            testID="sync-now"
          >
            <Text style={styles.manualButtonText}>
              {syncing ? "Syncing…" : "Sync now"}
            </Text>
          </Pressable>
        </View>
        <Text style={styles.who}>{user?.email}</Text>
      </View>

      {manualOpen && (
        <ManualEntry
          value={manualCode}
          onChange={setManualCode}
          onSubmit={handleManualSubmit}
          onCancel={() => setManualOpen(false)}
        />
      )}

      {outcome && (
        <ScanResultOverlay
          outcome={outcome}
          onDismiss={() => setOutcome(null)}
          onUndo={undo}
        />
      )}
    </View>
  );
}

function ManualEntry({
  value,
  onChange,
  onSubmit,
  onCancel,
}: {
  value: string;
  onChange: (next: string) => void;
  onSubmit: () => void;
  onCancel: () => void;
}) {
  return (
    <View style={styles.sheet}>
      <ScrollView keyboardShouldPersistTaps="handled">
        <Text style={styles.sheetTitle}>Enter the ticket code</Text>
        <Text style={styles.muted}>
          The QR token on the ticket, starting with TKT_. It is checked against
          the roster on this device.
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
          onSubmitEditing={onSubmit}
          returnKeyType="go"
        />
        <View style={styles.sheetActions}>
          <Pressable
            style={[styles.primaryButton, styles.primaryButtonInRow]}
            onPress={onSubmit}
          >
            <Text style={styles.primaryButtonText}>Check in</Text>
          </Pressable>
          <Pressable style={styles.secondaryButton} onPress={onCancel}>
            <Text style={styles.secondaryButtonText}>Cancel</Text>
          </Pressable>
        </View>
      </ScrollView>
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
  topBar: {
    position: "absolute",
    top: 0,
    left: 0,
    right: 0,
    flexDirection: "row",
    gap: 8,
    padding: 14,
    paddingTop: 54,
  },
  pill: { borderRadius: 999, paddingHorizontal: 12, paddingVertical: 6 },
  pillOnline: { backgroundColor: "rgba(21,128,61,0.9)" },
  pillOffline: { backgroundColor: "rgba(180,83,9,0.95)" },
  pillPending: { backgroundColor: "rgba(15,23,42,0.85)", borderColor: theme.brand, borderWidth: 1 },
  pillText: { color: "#fff", fontSize: 13, fontWeight: "700" },
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
  syncNote: { color: theme.brand, fontSize: 13, textAlign: "center", paddingHorizontal: 20 },
  buttonRow: { flexDirection: "row", gap: 10 },
  manualButton: {
    borderColor: "rgba(255,255,255,0.6)",
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: 18,
    paddingVertical: 10,
  },
  disabled: { opacity: 0.4 },
  manualButtonText: { color: "#fff", fontSize: 14, fontWeight: "600" },
  who: { color: "rgba(255,255,255,0.6)", fontSize: 12 },
  sheet: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    maxHeight: "70%",
    backgroundColor: theme.surface,
    borderTopLeftRadius: 22,
    borderTopRightRadius: 22,
    padding: 24,
    paddingBottom: 40,
    zIndex: 5,
  },
  sheetTitle: { color: theme.text, fontSize: 19, fontWeight: "700", marginBottom: 6 },
  sheetInput: {
    backgroundColor: theme.surfaceAlt,
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: 14,
    paddingVertical: 14,
    color: theme.text,
    fontSize: 15,
    marginTop: 10,
  },
  sheetActions: { flexDirection: "row", gap: 12, marginTop: 14 },
  primaryButton: {
    backgroundColor: theme.brandDark,
    borderRadius: radius.md,
    paddingVertical: 15,
    paddingHorizontal: 28,
    alignItems: "center",
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
