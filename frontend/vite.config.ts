import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";

const port = Number(process.env.WAILS_VITE_PORT) || 10245;

export default defineConfig({
  server: {
    host: "127.0.0.1",
    port,
    strictPort: true,
  },
  optimizeDeps: {
    include: ["vue", "ant-design-vue", "dayjs", "@wailsio/runtime"],
  },
  plugins: [vue(), tailwindcss(), wails("./bindings")],
});
