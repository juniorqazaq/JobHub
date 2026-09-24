import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { PropsWithChildren } from "react";
import { repositories } from "../api/repositories";
import type { AuthSession, LoginRequest, RegisterRequest } from "../api/models/auth";
import { AuthContext } from "./authContext";

export function AuthProvider({ children }: PropsWithChildren) {
  const queryClient = useQueryClient();
  const sessionQuery = useQuery({
    queryKey: ["auth", "me"],
    queryFn: () => repositories.auth.me(),
    retry: false,
  });
  const registerMutation = useMutation({
    mutationFn: (input: RegisterRequest) => repositories.auth.register(input),
    onSuccess: (session) => queryClient.setQueryData(["auth", "me"], session),
  });
  const loginMutation = useMutation({
    mutationFn: (input: LoginRequest) => repositories.auth.login(input),
    onSuccess: (session) => queryClient.setQueryData(["auth", "me"], session),
  });

  const logout = async () => {
    const session = queryClient.getQueryData<AuthSession>(["auth", "me"]);
    if (session?.csrfToken) await repositories.auth.logout(session.csrfToken);
    queryClient.setQueryData(["auth", "me"], undefined);
    await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
  };

  const session = sessionQuery.data;
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
