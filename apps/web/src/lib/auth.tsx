"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { UserManager, WebStorageStateStore, type User } from "oidc-client-ts";

const defaultOrigin =
  process.env.NODE_ENV === "production" && process.env.NEXT_PUBLIC_API_BASE_URL
    ? process.env.NEXT_PUBLIC_API_BASE_URL
    : "http://127.0.0.1:8080";

export const baseURL = (
  process.env.NEXT_PUBLIC_API_BASE_URL || defaultOrigin
).replace(/\/$/, "");

export const isLocalAPI =
  /^https?:\/\/(localhost|127\.0\.0\.1|\[::1\])(?::\d+)?$/i.test(baseURL);

export const cognitoClientId =
  process.env.NEXT_PUBLIC_COGNITO_CLIENT_ID ||
  (isLocalAPI ? "local-shortly-web" : "");

export const cognitoDomain = (
  process.env.NEXT_PUBLIC_COGNITO_DOMAIN ||
  (isLocalAPI ? `${baseURL}/oauth2-local` : "")
).replace(/\/$/, "");

export const cognitoIssuer = (
  process.env.NEXT_PUBLIC_COGNITO_ISSUER ||
  (isLocalAPI ? `${baseURL}/oauth2-local` : "")
).replace(/\/$/, "");

export const requiredScopes =
  "openid email shortly-api/links.read shortly-api/links.write";

export type AuthSession = {
  sub: string;
  email?: string;
  scopes: string[];
  expiresAt?: number;
};

type AuthContextValue = {
  session: AuthSession | null;
  loading: boolean;
  authError: string;
  clearAuthError: () => void;
  signIn: (options?: {
    returnTo?: string;
    loginHint?: string;
  }) => Promise<void>;
  signUp: (options?: { returnTo?: string }) => Promise<void>;
  signOut: () => Promise<void>;
  refreshSession: () => Promise<string | null>;
};

let managerInstance: UserManager | null = null;
const authListeners = new Set<() => void>();
let lastSessionError = "";

function notifyAuthListeners() {
  for (const listener of authListeners) {
    listener();
  }
}

export function setAuthErrorNotice(message: string) {
  lastSessionError = message;
  notifyAuthListeners();
}

export function getUserManager(): UserManager | null {
  if (typeof window === "undefined") return null;
  if (!cognitoClientId || !cognitoDomain || !cognitoIssuer) return null;
  if (!managerInstance) {
    const origin = window.location.origin;
    const store = new WebStorageStateStore({ store: window.sessionStorage });
    managerInstance = new UserManager({
      authority: cognitoIssuer,
      client_id: cognitoClientId,
      redirect_uri: `${origin}/auth/callback/`,
      post_logout_redirect_uri: `${origin}/`,
      response_type: "code",
      scope: requiredScopes,
      userStore: store,
      stateStore: store,
      automaticSilentRenew: true,
      loadUserInfo: false,
      metadata: {
        issuer: cognitoIssuer,
        authorization_endpoint: `${cognitoDomain}/oauth2/authorize`,
        token_endpoint: `${cognitoDomain}/oauth2/token`,
        userinfo_endpoint: `${cognitoDomain}/oauth2/userInfo`,
        end_session_endpoint: `${cognitoDomain}/logout`,
        revocation_endpoint: `${cognitoDomain}/oauth2/revoke`,
        jwks_uri: `${cognitoIssuer}/.well-known/jwks.json`,
      },
    });
    managerInstance.events.addUserLoaded(() => {
      lastSessionError = "";
      notifyAuthListeners();
    });
    managerInstance.events.addUserUnloaded(() => {
      notifyAuthListeners();
    });
    managerInstance.events.addSilentRenewError(() => {
      lastSessionError = "Your session has expired. Please sign in again.";
      notifyAuthListeners();
    });
  }
  return managerInstance;
}

function toAuthSession(user: User | null): AuthSession | null {
  if (!user || !user.access_token || user.expired) return null;
  const sub = typeof user.profile?.sub === "string" ? user.profile.sub : "";
  if (!sub) return null;
  const email =
    typeof user.profile?.email === "string" ? user.profile.email : undefined;
  return {
    sub,
    email,
    scopes: user.scopes || [],
    expiresAt: user.expires_at,
  };
}

export async function forceRefreshAccessToken(): Promise<string | null> {
  const manager = getUserManager();
  if (!manager) return null;
  try {
    const existing = await manager.getUser();
    if (!existing?.refresh_token) {
      if (existing) {
        await manager.removeUser();
        setAuthErrorNotice("Your session has expired. Please sign in again.");
      }
      return null;
    }
    const refreshed = await manager.signinSilent();
    if (!refreshed?.access_token) {
      await manager.removeUser();
      setAuthErrorNotice("Your session has expired. Please sign in again.");
      return null;
    }
    lastSessionError = "";
    notifyAuthListeners();
    return refreshed.access_token;
  } catch {
    await manager.removeUser().catch(() => {});
    setAuthErrorNotice("Your session has expired. Please sign in again.");
    return null;
  }
}

export async function getValidAccessToken(): Promise<string | null> {
  const manager = getUserManager();
  if (!manager) return null;
  const user = await manager.getUser().catch(() => null);
  if (!user) return null;
  if (user.expired) {
    return forceRefreshAccessToken();
  }
  return user.access_token || null;
}

export async function completeAuthCallback(
  url: string,
): Promise<{ returnTo: string }> {
  await Promise.resolve();
  const parsed = new URL(url);
  const oauthError = parsed.searchParams.get("error");
  const oauthDescription = parsed.searchParams.get("error_description");
  if (oauthError) {
    throw new Error(
      oauthDescription || `Sign-in was not completed (${oauthError}).`,
    );
  }
  const manager = getUserManager();
  if (!manager) {
    throw new Error("Authentication client is not configured.");
  }
  const user = await manager.signinRedirectCallback(url);
  lastSessionError = "";
  notifyAuthListeners();
  const state = user.state as { returnTo?: string } | undefined;
  const returnTo =
    state?.returnTo && state.returnTo.startsWith("/")
      ? state.returnTo
      : "/dashboard/";
  return { returnTo };
}

function navigateToHostedUI(url: string) {
  if (typeof window !== "undefined") {
    window.location.assign(url);
  }
}

async function resolveCurrentUser(): Promise<User | null> {
  const manager = getUserManager();
  if (!manager) return null;
  let user = await manager.getUser();
  if (user?.expired && user.refresh_token) {
    await forceRefreshAccessToken();
    user = await manager.getUser();
  } else if (user?.expired) {
    await manager.removeUser();
    user = null;
    lastSessionError = "Your session has expired. Please sign in again.";
  }
  return user;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<AuthSession | null>(null);
  const [loading, setLoading] = useState(true);
  const [authError, setAuthError] = useState("");

  const syncUser = useCallback(() => {
    return resolveCurrentUser()
      .then((user) => {
        setSession(toAuthSession(user));
        setAuthError(lastSessionError);
      })
      .catch(() => {
        setSession(null);
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  useEffect(() => {
    const listener = () => {
      void syncUser();
    };
    authListeners.add(listener);
    void syncUser();
    return () => {
      authListeners.delete(listener);
    };
  }, [syncUser]);

  const signIn = useCallback(
    async (options?: { returnTo?: string; loginHint?: string }) => {
      const manager = getUserManager();
      if (!manager) {
        setAuthErrorNotice("Cognito authentication is not configured.");
        return;
      }
      lastSessionError = "";
      setAuthError("");
      await manager.signinRedirect({
        state: {
          returnTo:
            options?.returnTo ||
            (typeof window !== "undefined" ? window.location.pathname : "/"),
        },
        ...(options?.loginHint
          ? { extraQueryParams: { login_hint: options.loginHint } }
          : {}),
      });
    },
    [],
  );

  const signUp = useCallback(async (options?: { returnTo?: string }) => {
    const manager = getUserManager();
    if (!manager) {
      setAuthErrorNotice("Cognito authentication is not configured.");
      return;
    }
    lastSessionError = "";
    setAuthError("");
    await manager.signinRedirect({
      state: {
        returnTo:
          options?.returnTo ||
          (typeof window !== "undefined" ? window.location.pathname : "/"),
      },
      extraQueryParams: {
        screen_hint: "signup",
      },
    });
  }, []);

  const signOut = useCallback(async () => {
    const manager = getUserManager();
    lastSessionError = "";
    setAuthError("");
    if (!manager) {
      setSession(null);
      return;
    }
    await manager.revokeTokens(["refresh_token"]).catch(() => {});
    await manager.removeUser().catch(() => {});
    setSession(null);
    notifyAuthListeners();
    if (typeof window !== "undefined" && cognitoDomain && !isLocalAPI) {
      const logoutURI = `${window.location.origin}/`;
      navigateToHostedUI(
        `${cognitoDomain}/logout?client_id=${encodeURIComponent(cognitoClientId)}&logout_uri=${encodeURIComponent(logoutURI)}`,
      );
    }
  }, []);

  const refreshSession = useCallback(async () => {
    const token = await forceRefreshAccessToken();
    await syncUser();
    return token;
  }, [syncUser]);

  const clearAuthError = useCallback(() => {
    lastSessionError = "";
    setAuthError("");
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      session,
      loading,
      authError,
      clearAuthError,
      signIn,
      signUp,
      signOut,
      refreshSession,
    }),
    [
      session,
      loading,
      authError,
      clearAuthError,
      signIn,
      signUp,
      signOut,
      refreshSession,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
