import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import { AppProviders } from "./app/AppProviders";
import "@fontsource-variable/manrope";
import "@fontsource-variable/source-sans-3";
import "./styles/index.css";
import "./components/ui/ui.css";
import "./pages/component-showcase.css";
import "./styles/marketplace.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <AppProviders>
        <App />
      </AppProviders>
    </BrowserRouter>
  </React.StrictMode>,
);
