import { zodResolver } from "@hookform/resolvers/zod";
import { BriefcaseBusiness, Eye, EyeOff, UserRound } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { z } from "zod";
import { useAuth } from "../app/authContext";
import type { UserRole } from "../api/models/auth";
import { Button, IconButton } from "../components/ui/Button";
import { Input } from "../components/ui/FormControls";
import { SiteShell } from "../layouts/SiteShell";
import { safeReturnTo } from "../lib/returnTo";
import { authErrorCode } from "./authUtils";

const schema = z.object({
  fullName: z.string().trim().min(1),
  email: z.email(),
  password: z.string().min(10),
  confirmPassword: z.string().min(1),
  companyName: z.string().optional(),
}).refine((values) => values.password === values.confirmPassword, { path: ["confirmPassword"] });

type RegisterValues = z.infer<typeof schema>;

export function RegisterPage() {
  const { t } = useTranslation();
  const auth = useAuth();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [role, setRole] = useState<Extract<UserRole, "job_seeker" | "employer">>("job_seeker");
  const [showPassword, setShowPassword] = useState(false);
  const [serverError, setServerError] = useState("");
  const { register, handleSubmit, formState } = useForm<RegisterValues>({
    resolver: zodResolver(schema),
    defaultValues: { fullName: "", email: "", password: "", confirmPassword: "", companyName: "" },
  });

  const onSubmit = handleSubmit(async (values) => {
    setServerError("");
    if (role === "employer" && !values.companyName?.trim()) {
      setServerError(t("auth.validation.companyRequired"));
      return;
    }
    try {
      await auth.register({ fullName: values.fullName, email: values.email, password: values.password, role, companyName: values.companyName });
      navigate(role === "employer" ? "/employer" : safeReturnTo(params.get("returnTo")), { replace: true });
    } catch (error) {
      const code = authErrorCode(error);
      setServerError(code === "EMAIL_ALREADY_EXISTS" ? t("auth.errors.emailExists") : t("auth.errors.generic"));
    }
  });

  return (
    <SiteShell>
      <main className="auth-page page-container">
        <section className="auth-panel auth-panel--wide" aria-labelledby="register-title">
          <p className="auth-eyebrow">{t("auth.registerEyebrow")}</p>
          <h1 id="register-title">{t("auth.registerTitle")}</h1>
          <div className="role-picker" role="radiogroup" aria-label={t("auth.chooseRole")}>
            <button type="button" className={role === "job_seeker" ? "active" : ""} onClick={() => setRole("job_seeker")}><UserRound size={20} /><span>{t("auth.roles.job_seeker")}</span></button>
            <button type="button" className={role === "employer" ? "active" : ""} onClick={() => setRole("employer")}><BriefcaseBusiness size={20} /><span>{t("auth.roles.employer")}</span></button>
          </div>
          <form className="auth-form" onSubmit={onSubmit} noValidate>
            <Input label={t("auth.fullName")} autoComplete="name" error={formState.errors.fullName ? t("auth.validation.fullName") : undefined} {...register("fullName")} />
            <Input label={role === "employer" ? t("auth.workEmail") : t("auth.email")} type="email" autoComplete="email" error={formState.errors.email ? t("auth.validation.email") : undefined} {...register("email")} />
            {role === "employer" ? <Input label={t("auth.companyName")} autoComplete="organization" error={serverError === t("auth.validation.companyRequired") ? serverError : undefined} {...register("companyName")} /> : null}
            <div className="password-field">
              <Input label={t("auth.password")} type={showPassword ? "text" : "password"} autoComplete="new-password" error={formState.errors.password ? t("auth.validation.passwordPolicy") : undefined} {...register("password")} />
              <IconButton type="button" label={showPassword ? t("auth.hidePassword") : t("auth.showPassword")} icon={showPassword ? <EyeOff size={18} /> : <Eye size={18} />} variant="quiet" onClick={() => setShowPassword((value) => !value)} />
            </div>
            <Input label={t("auth.confirmPassword")} type={showPassword ? "text" : "password"} autoComplete="new-password" error={formState.errors.confirmPassword ? t("auth.validation.passwordMatch") : undefined} {...register("confirmPassword")} />
            {serverError && serverError !== t("auth.validation.companyRequired") ? <p className="auth-error" role="alert">{serverError}</p> : null}
            <Button type="submit" isLoading={formState.isSubmitting}>{t("auth.registerSubmit")}</Button>
          </form>
          <p className="auth-switch">{t("auth.hasAccount")} <Link to={`/login?returnTo=${encodeURIComponent(safeReturnTo(params.get("returnTo")))}`}>{t("auth.loginLink")}</Link></p>
        </section>
      </main>
    </SiteShell>
  );
}
