import globals from "globals";
import pluginJs from "@eslint/js";
import tseslint from "typescript-eslint";
import pluginReact from "eslint-plugin-react";

/** @type {import('eslint').Linter.Config[]} */
export default [
  {
    // Ignores has to be in a separate object from "files",
    // or it will only exclude files matched by "files".
    ignores: [
      "eslint.config.mjs",
      "tailwind.config.js",
      "prettier.config.mjs",
      "gen/**",
      "dist/index.js",
    ],
  },
  {
    languageOptions: {
      globals: globals.browser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
        ecmaFeatures: {
          jsx: true,
        },
      },
    },
    settings: {
      react: {
        version: "detect",
        defaultVersion: "19.0.0",
      },
    },
  },
  {
    files: ["components/**/*.tsx", "pages/**/*.tsx", "index.tsx", "App.tsx"],
  },
  pluginJs.configs.recommended,
  pluginReact.configs.flat.recommended,
  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,
  {
    rules: {
      eqeqeq: "error",
    },
  },
];
