import type { ReactElement } from "react";
import React from "react";

// Based on https://www.paulie.dev/posts/2023/11/a-set-of-sign-in-with-google-buttons-made-with-tailwind/
/*
MIT License

Copyright (c) 2023 Paul Scanlon

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

interface SignInButtonProps {
  logo: ReactElement<SVGSVGElement>;
  text: string;
}

const SignInButton: React.FC<SignInButtonProps> = ({ logo, text }) => (
  <div
    aria-label={text}
    className="flex items-center justify-center bg-white border rounded-md p-0.5 pr-3 hover:shadow-md"
  >
    <div className="flex items-center justify-center bg-white w-9 h-9 rounded-l p-1">
      {logo}
    </div>
    <span className="text-sm text-google-text-gray tracking-wider">{text}</span>
  </div>
);

export default SignInButton;
