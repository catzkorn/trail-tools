import type { Config } from "tailwindcss";

export default {
  content: ["./dist/index.{js,html}"],
  theme: {
    extend: {},
  },
  plugins: [],
} satisfies Config;
