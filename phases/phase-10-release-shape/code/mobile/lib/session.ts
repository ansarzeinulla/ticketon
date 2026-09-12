import * as SecureStore from "expo-secure-store";

/**
 * The access token lives in the device keychain / keystore, not in plain
 * AsyncStorage. A scanner is a shared device that sits on a table at a venue
 * entrance, so the credential has to be protected at rest.
 */
const TOKEN_KEY = "biletflow.access_token";
const EMAIL_KEY = "biletflow.email";

export async function saveSession(token: string, email: string): Promise<void> {
  await SecureStore.setItemAsync(TOKEN_KEY, token);
  await SecureStore.setItemAsync(EMAIL_KEY, email);
}

export async function loadToken(): Promise<string | null> {
  try {
    return await SecureStore.getItemAsync(TOKEN_KEY);
  } catch {
    return null;
  }
}

export async function loadEmail(): Promise<string | null> {
  try {
    return await SecureStore.getItemAsync(EMAIL_KEY);
  } catch {
    return null;
  }
}

export async function clearSession(): Promise<void> {
  await SecureStore.deleteItemAsync(TOKEN_KEY);
  await SecureStore.deleteItemAsync(EMAIL_KEY);
}
