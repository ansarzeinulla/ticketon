import { Redirect } from "expo-router";
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";

import { useAuth } from "../lib/auth-context";
import { radius, theme } from "../lib/theme";

/**
 * The first screen: sign-in when there is no session, otherwise who is signed
 * in. The event list takes this screen's place once the API can list events.
 */
export default function Index() {
  const { status, user, signOut } = useAuth();

  if (status === "loading") {
    return (
      <View style={styles.container}>
        <ActivityIndicator color={theme.brand} size="large" />
        <Text style={styles.label}>Restoring your session…</Text>
      </View>
    );
  }

  if (status === "signedOut") return <Redirect href="/login" />;

  return (
    <View style={styles.container}>
      <Text style={styles.heading}>Signed in</Text>
      <Text style={styles.label}>{user?.full_name}</Text>
      <Text style={styles.label}>{user?.email}</Text>
      <Pressable style={styles.button} onPress={() => void signOut()} testID="sign-out">
        <Text style={styles.buttonText}>Sign out</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 16,
    padding: 24,
    backgroundColor: theme.bg,
  },
  heading: { color: theme.text, fontSize: 24, fontWeight: "700" },
  label: { color: theme.textMuted, fontSize: 15 },
  button: {
    marginTop: 16,
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingVertical: 12,
    paddingHorizontal: 24,
  },
  buttonText: { color: theme.text, fontSize: 15, fontWeight: "600" },
});
