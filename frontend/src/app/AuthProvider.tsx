import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { PropsWithChildren } from "react";
import { repositories } from "../api/repositories";
import type { AuthSession, LoginRequest, RegisterRequest } from "../api/models/auth";
import { AuthContext } from "./authContext";

export function AuthProvider({ children }: PropsWithChildren) {
  const queryClient = useQueryClient();
  const sessionQuery = useQuery<AuthSession | null>({
    queryKey: ["auth", "me"],
    queryFn: () => repositories.auth.me(),
    retry: false,
  });
  const registerMutation = useMutation({
    mutationFn: (input: RegisterRequest) => repositories.auth.register(input),
    onSuccess: (session) => queryClient.setQueryData<AuthSession | null>(["auth", "me"], session),
  });
  const loginMutation = useMutation({
    mutationFn: (input: LoginRequest) => repositories.auth.login(input),
    onSuccess: (session) => queryClient.setQueryData<AuthSession | null>(["auth", "me"], session),
  });

  const logout = async () => {
    const session = queryClient.getQueryData<AuthSession>(["auth", "me"]);
    if (session?.csrfToken) await repositories.auth.logout(session.csrfToken);
    queryClient.setQueryData<AuthSession | null>(["auth", "me"], null);
  };

  const session = sessionQuery.data ?? undefined;
  return (
    <AuthContext.Provider
      value={{
        session,
        isLoading: sessionQuery.isPending,
        isAuthenticated: Boolean(session),
        register: registerMutation.mutateAsync,
        login: loginMutation.mutateAsync,
        logout,
        hasRole: (role) => session?.user.role === role,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}
