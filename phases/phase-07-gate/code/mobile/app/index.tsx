import { Redirect } from "expo-router";
import { ActivityIndicator, StyleSheet, Text, View } from "react-native";

import { useAuth } from "../lib/auth-context";
import { theme } from "../lib/theme";

/** Sends the app to sign-in or to the event selector once the session settles. */
export default function Index() {
  const { status } = useAuth();

  if (status === "loading") {
    return (
      <View style={styles.container}>
        <ActivityIndicator color={theme.brand} size="large" />
        <Text style={styles.label}>Restoring your session…</Text>
      </View>
    );
  }

  return <Redirect href={status === "signedIn" ? "/events" : "/login"} />;
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 16,
    backgroundColor: theme.bg,
  },
  label: { color: theme.textMuted, fontSize: 15 },
});
