import type { Config } from "tailwindcss";

export default {
  content: ["./dist/index.{js,html}"],
  theme: {
    extend: {
      colors: {
        google: {
          "text-gray": "#3c4043",
          "logo-blue": "#4285f4",
          "logo-green": "#34a853",
          "logo-yellow": "#fbbc05",
          "logo-red": "#ea4335",
        },
      },
    },
  },
  plugins: [],
} satisfies Config;
