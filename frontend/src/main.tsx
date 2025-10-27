import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import App from "./App";

const rootElement = document.getElementById("root");

if (!rootElement) {
  throw new Error("未找到 #root 根节点，请检查 index.html 的结构。");
}

const root = ReactDOM.createRoot(rootElement);
const queryClient = new QueryClient();

root.render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
