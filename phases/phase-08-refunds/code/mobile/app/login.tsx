import { router } from "expo-router";
import { useEffect, useState } from "react";
import {
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { ApiError } from "../lib/api";
import { useAuth } from "../lib/auth-context";
import { API_HOST_LABEL } from "../lib/config";
import { radius, theme } from "../lib/theme";

export default function LoginScreen() {
  const { signIn, status, lastEmail } = useAuth();
  const insets = useSafeAreaInsets();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Venue staff sign in on a shared device shift after shift; pre-filling the
  // last address saves retyping it every time.
  useEffect(() => {
    if (lastEmail) setEmail(lastEmail);
  }, [lastEmail]);

  useEffect(() => {
    if (status === "signedIn") router.replace("/events");
  }, [status]);

  async function handleSignIn() {
    setError(null);

    if (!email.trim() || !password) {
      setError("Enter your email and password.");
      return;
    }

    setSubmitting(true);
    try {
      await signIn(email.trim().toLowerCase(), password);
      router.replace("/events");
    } catch (cause) {
      if (cause instanceof ApiError) {
        setError(
          cause.code === "invalid_credentials"
            ? "Email or password is incorrect."
            : cause.message,
        );
      } else {
        setError("Sign-in failed. Please try again.");
      }
      setSubmitting(false);
    }
  }

  return (
    <KeyboardAvoidingView
      style={styles.flex}
      behavior={Platform.OS === "ios" ? "padding" : undefined}
    >
      <ScrollView
        contentContainerStyle={[styles.container, { paddingTop: insets.top + 48 }]}
        keyboardShouldPersistTaps="handled"
      >
        <View style={styles.brandRow}>
          <View style={styles.logo}>
            <Text style={styles.logoText}>B</Text>
          </View>
          <View>
            <Text style={styles.brand}>BiletFlow</Text>
            <Text style={styles.brandSub}>Ticket scanner</Text>
          </View>
        </View>

        <Text style={styles.heading}>Event Admin sign-in</Text>
        <Text style={styles.subheading}>
          Use the BiletFlow account the organizer assigned to this event.
        </Text>

        {error && (
          <View style={styles.errorBox} accessibilityLiveRegion="polite">
            <Text style={styles.errorText}>{error}</Text>
          </View>
        )}

        <Text style={styles.label}>Email</Text>
        <TextInput
          style={styles.input}
          value={email}
          onChangeText={setEmail}
          placeholder="scanner@biletflow.kz"
          placeholderTextColor={theme.textMuted}
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="email-address"
          textContentType="emailAddress"
          editable={!submitting}
          testID="login-email"
        />

        <Text style={styles.label}>Password</Text>
        <TextInput
          style={styles.input}
          value={password}
          onChangeText={setPassword}
          placeholder="••••••••"
          placeholderTextColor={theme.textMuted}
          secureTextEntry
          autoCapitalize="none"
          textContentType="password"
          editable={!submitting}
          onSubmitEditing={handleSignIn}
          returnKeyType="go"
          testID="login-password"
        />

        <Pressable
          style={({ pressed }) => [
            styles.button,
            (pressed || submitting) && styles.buttonPressed,
          ]}
          onPress={handleSignIn}
          disabled={submitting}
          testID="login-submit"
        >
          {submitting ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.buttonText}>Sign in</Text>
          )}
        </Pressable>

        {/* Pointing at the wrong machine is the most common setup mistake, so
            the target is shown rather than hidden in a config file. */}
        <Text style={styles.host}>API: {API_HOST_LABEL}</Text>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, backgroundColor: theme.bg },
  container: { padding: 24, paddingBottom: 48, gap: 4 },
  brandRow: { flexDirection: "row", alignItems: "center", gap: 12, marginBottom: 40 },
  logo: {
    width: 44,
    height: 44,
    borderRadius: radius.md,
    backgroundColor: theme.brandDark,
    alignItems: "center",
    justifyContent: "center",
  },
  logoText: { color: "#fff", fontSize: 22, fontWeight: "700" },
  brand: { color: theme.text, fontSize: 20, fontWeight: "700" },
  brandSub: { color: theme.textMuted, fontSize: 13 },
  heading: { color: theme.text, fontSize: 24, fontWeight: "700" },
  subheading: { color: theme.textMuted, fontSize: 14, marginTop: 6, marginBottom: 24 },
  label: { color: theme.textMuted, fontSize: 13, marginTop: 16, marginBottom: 6 },
  input: {
    backgroundColor: theme.surface,
    borderColor: theme.border,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: 14,
    paddingVertical: 14,
    color: theme.text,
    fontSize: 16,
  },
  button: {
    marginTop: 28,
    backgroundColor: theme.brandDark,
    borderRadius: radius.md,
    paddingVertical: 16,
    alignItems: "center",
  },
  buttonPressed: { opacity: 0.75 },
  buttonText: { color: "#fff", fontSize: 16, fontWeight: "600" },
  errorBox: {
    backgroundColor: "#450a0a",
    borderColor: theme.danger,
    borderWidth: 1,
    borderRadius: radius.md,
    padding: 12,
    marginBottom: 8,
  },
  errorText: { color: "#fecaca", fontSize: 14 },
  host: { color: theme.textMuted, fontSize: 12, textAlign: "center", marginTop: 24 },
});
