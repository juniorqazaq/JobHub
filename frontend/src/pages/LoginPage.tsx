import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, EyeOff } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { z } from "zod";
import { useAuth } from "../app/authContext";
import { Button, IconButton } from "../components/ui/Button";
import { Input } from "../components/ui/FormControls";
import { SiteShell } from "../layouts/SiteShell";
import { safeReturnTo } from "../lib/returnTo";
import { authErrorCode } from "./authUtils";

const schema = z.object({
  email: z.email(),
  password: z.string().min(1),
});

type LoginValues = z.infer<typeof schema>;

export function LoginPage() {
  const { t } = useTranslation();
  const auth = useAuth();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [showPassword, setShowPassword] = useState(false);
  const [serverError, setServerError] = useState("");
  const { register, handleSubmit, formState } = useForm<LoginValues>({ resolver: zodResolver(schema), defaultValues: { email: "", password: "" } });

  const onSubmit = handleSubmit(async (values) => {
    setServerError("");
    try {
      await auth.login(values);
      navigate(safeReturnTo(params.get("returnTo")), { replace: true });
    } catch (error) {
      const code = authErrorCode(error);
      setServerError(code === "INVALID_CREDENTIALS" ? t("auth.errors.invalidCredentials") : t("auth.errors.generic"));
    }
  });

  return (
    <SiteShell>
      <main className="auth-page page-container">
        <section className="auth-panel" aria-labelledby="login-title">
          <p className="auth-eyebrow">{t("auth.loginEyebrow")}</p>
          <h1 id="login-title">{t("auth.loginTitle")}</h1>
          <form className="auth-form" onSubmit={onSubmit} noValidate>
            <Input label={t("auth.email")} type="email" autoComplete="email" error={formState.errors.email?.message ? t("auth.validation.email") : undefined} {...register("email")} />
            <div className="password-field">
              <Input label={t("auth.password")} type={showPassword ? "text" : "password"} autoComplete="current-password" error={formState.errors.password?.message ? t("auth.validation.passwordRequired") : undefined} {...register("password")} />
              <IconButton type="button" label={showPassword ? t("auth.hidePassword") : t("auth.showPassword")} icon={showPassword ? <EyeOff size={18} /> : <Eye size={18} />} variant="quiet" onClick={() => setShowPassword((value) => !value)} />
            </div>
            {serverError ? <p className="auth-error" role="alert">{serverError}</p> : null}
            <Button type="submit" isLoading={formState.isSubmitting}>{t("auth.loginSubmit")}</Button>
          </form>
          <p className="auth-switch">{t("auth.noAccount")} <Link to={`/register?returnTo=${encodeURIComponent(safeReturnTo(params.get("returnTo")))}`}>{t("auth.registerLink")}</Link></p>
        </section>
      </main>
    </SiteShell>
  );
}
