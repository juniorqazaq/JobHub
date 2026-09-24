import { lazy, Suspense, useEffect } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";

const HomePage = lazy(() => import("./pages/HomePage").then((module) => ({ default: module.HomePage })));
const JobsPage = lazy(() => import("./pages/JobsPage").then((module) => ({ default: module.JobsPage })));
const JobDetailPage = lazy(() => import("./pages/JobDetailPage").then((module) => ({ default: module.JobDetailPage })));
const LoginPage = lazy(() => import("./pages/LoginPage").then((module) => ({ default: module.LoginPage })));
const RegisterPage = lazy(() => import("./pages/RegisterPage").then((module) => ({ default: module.RegisterPage })));
const AccountPage = lazy(() => import("./pages/AccountPage").then((module) => ({ default: module.AccountPage })));
const EmployerPage = lazy(() => import("./pages/EmployerPage").then((module) => ({ default: module.EmployerPage })));
const AdminPage = lazy(() => import("./pages/AdminPage").then((module) => ({ default: module.AdminPage })));
const ProtectedRoute = lazy(() => import("./pages/ProtectedRoute").then((module) => ({ default: module.ProtectedRoute })));

const ComponentShowcasePage = lazy(() =>
  import("./pages/ComponentShowcasePage").then((module) => ({
    default: module.ComponentShowcasePage,
  })),
);

export default function App() {
  return (
    <>
      <ScrollToTop />
      <Suspense fallback={<div className="route-loading" aria-hidden="true" />}>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/jobs" element={<JobsPage />} />
          <Route path="/jobs/:jobId" element={<JobDetailPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/account" element={<ProtectedRoute><AccountPage /></ProtectedRoute>} />
          <Route path="/employer" element={<ProtectedRoute role="employer"><EmployerPage /></ProtectedRoute>} />
          <Route path="/admin" element={<ProtectedRoute role="admin"><AdminPage /></ProtectedRoute>} />
          <Route path="/dev/ui" element={<ComponentShowcasePage />} />
          <Route path="/dev/components" element={<Navigate to="/dev/ui" replace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
    </>
  );
}

function ScrollToTop() {
  const { pathname } = useLocation();
  useEffect(() => {
    window.scrollTo({ top: 0, left: 0 });
  }, [pathname]);
  return null;
}
