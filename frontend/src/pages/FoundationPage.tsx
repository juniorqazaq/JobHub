import { useQuery } from "@tanstack/react-query";
import { getHealth } from "../api/health";
export function FoundationPage() {
  const health = useQuery({ queryKey: ["health"], queryFn: getHealth });
  const ready = health.isSuccess && health.data.status === "ok";
  return (
    <div className="shell">
      <header>
        <a className="brand" href="/">
          <span className="brand-icon">J</span> JobHub{" "}
          <span className="ai">AI</span>
        </a>
        <span className="phase">PHASE 01 / FOUNDATION</span>
      </header>
      <main>
        <section className="hero">
          <div className="eyebrow">
            <span /> A smarter job search starts here
          </div>
          <h1>
            One place.
            <br />
            More possibilities.
          </h1>
          <p className="intro">
            The foundation for finding work that fits you. Bringing
            opportunities, your skills, and your next chapter together.
          </p>
          <div className="hero-footer">
            <span className="pill">Development workspace</span>
            <span>Built with purpose. One phase at a time.</span>
          </div>
        </section>
        <section className="status-panel" aria-labelledby="status-title">
          <div className="panel-heading">
            <div>
              <span className="label">SYSTEM OVERVIEW</span>
              <h2 id="status-title">A solid starting point.</h2>
            </div>
            <button
              onClick={() => void health.refetch()}
              disabled={health.isFetching}
            >
              {health.isFetching ? "Checking…" : "↻ Check connection"}
            </button>
          </div>
          <div className="status-grid">
            <article>
              <span className="number">01</span>
              <h3>Frontend</h3>
              <p>React + TypeScript + Vite</p>
              <span className="state good">● Running in your browser</span>
            </article>
            <article>
              <span className="number">02</span>
              <h3>API & database</h3>
              <p>Go + Gin + PostgreSQL</p>
              <span
                role="status"
                className={"state " + (ready ? "good" : "pending")}
              >
                {health.isFetching
                  ? "◌ Checking connection"
                  : ready
                    ? "● Connected and healthy"
                    : "○ Connection unavailable"}
              </span>
            </article>
            <article>
              <span className="number">03</span>
              <h3>Infrastructure</h3>
              <p>Docker Compose + migrations</p>
              <span className="state neutral">◇ Configuration included</span>
            </article>
          </div>
          {health.isError && (
            <div className="notice" role="alert">
              The API is not reachable or its database is unavailable. Start the
              services with <code>docker compose up --build</code>, then check
              the connection again.
            </div>
          )}
        </section>
        <section className="next">
          <div>
            <span className="label">WHAT COMES NEXT</span>
            <h2>Make it yours.</h2>
          </div>
          <p>
            Phase 2 introduces accounts, secure sign-in, and protected routes.
            Job search and resume matching will follow in later phases.
          </p>
          <span className="next-mark">↗</span>
        </section>
      </main>
      <footer>
        <span>
          JobHub AI{" "}
          <span className="muted">/ From possibility to opportunity.</span>
        </span>
        <span>Foundation release · 0.1.0</span>
      </footer>
    </div>
  );
}
