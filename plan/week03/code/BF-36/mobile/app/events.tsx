import { router, useFocusEffect } from "expo-router";
import { useCallback, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from "react-native";

import { ApiError, api } from "../lib/api";
import { useAuth } from "../lib/auth-context";
import { radius, theme } from "../lib/theme";
import type { MobileEvent } from "../lib/types";

/** Renders a start time in the event's own timezone, not the phone's. */
function formatStart(event: MobileEvent): string {
  try {
    return new Intl.DateTimeFormat("en-GB", {
      timeZone: event.timezone,
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(event.starts_at));
  } catch {
    return new Date(event.starts_at).toLocaleString();
  }
}

export default function EventsScreen() {
  const { user, signOut } = useAuth();

  const [events, setEvents] = useState<MobileEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setEvents(await api.myPublishedEvents());
      setError(null);
    } catch (cause) {
      if (cause instanceof ApiError && cause.isSessionExpired) {
        await signOut();
        router.replace("/login");
        return;
      }
      setError(cause instanceof ApiError ? cause.message : "Could not load your events.");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [signOut]);

  // Re-fetch whenever the screen comes back into view.
  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  async function handleSignOut() {
    await signOut();
    router.replace("/login");
  }

  if (loading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator color={theme.brand} size="large" />
        <Text style={styles.muted}>Loading your events…</Text>
      </View>
    );
  }

  return (
    <View style={styles.flex}>
      <View style={styles.header}>
        <View style={styles.headerText}>
          <Text style={styles.who}>{user?.full_name}</Text>
          <Text style={styles.muted}>{user?.email}</Text>
        </View>
        <Pressable onPress={handleSignOut} hitSlop={12} testID="sign-out">
          <Text style={styles.signOut}>Sign out</Text>
        </Pressable>
      </View>

      {error && (
        <View style={styles.errorBox}>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      )}

      <FlatList
        data={events}
        keyExtractor={(item) => item.id}
        contentContainerStyle={styles.list}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={() => {
              setRefreshing(true);
              void load();
            }}
            tintColor={theme.brand}
          />
        }
        ListEmptyComponent={
          <View style={styles.empty}>
            <Text style={styles.emptyTitle}>No published events</Text>
            <Text style={styles.muted}>
              Publish an event from the web dashboard and it appears here. Pull
              down to refresh.
            </Text>
          </View>
        }
        renderItem={({ item }) => (
          <View style={styles.card} testID={`event-${item.id}`}>
            <Text style={styles.cardTitle} numberOfLines={2}>
              {item.title}
            </Text>
            <Text style={styles.muted}>{formatStart(item)}</Text>
            {item.venue_name ? <Text style={styles.muted}>{item.venue_name}</Text> : null}
          </View>
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: theme.bg },
  centered: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 14,
    backgroundColor: theme.bg,
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderBottomColor: theme.border,
    borderBottomWidth: 1,
  },
  headerText: { flexShrink: 1 },
  who: { color: theme.text, fontSize: 15, fontWeight: "600" },
  signOut: { color: theme.brand, fontSize: 14, fontWeight: "600" },
  muted: { color: theme.textMuted, fontSize: 13 },
  list: { padding: 16, gap: 12, flexGrow: 1 },
  card: {
    backgroundColor: theme.surface,
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: radius.lg,
    padding: 18,
    gap: 4,
  },
  cardTitle: { color: theme.text, fontSize: 17, fontWeight: "700", flex: 1 },
  empty: { flex: 1, alignItems: "center", justifyContent: "center", padding: 32, gap: 10 },
  emptyTitle: { color: theme.text, fontSize: 17, fontWeight: "600" },
  errorBox: {
    margin: 16,
    backgroundColor: "#450a0a",
    borderColor: theme.danger,
    borderWidth: 1,
    borderRadius: radius.md,
    padding: 12,
  },
  errorText: { color: "#fecaca", fontSize: 14 },
});
