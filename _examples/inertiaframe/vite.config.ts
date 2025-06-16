import { defineConfig } from "vite";
import react from "@vitejs/plugin-react-swc";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    manifest: true,
    outDir: "build",
    assetsDir: "",
    emptyOutDir: true,
    rollupOptions: {
      input: "./src/main.tsx",
    },
  },
});
