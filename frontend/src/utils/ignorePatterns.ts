/** Escape regex metacharacters except glob wildcards * and ? */
function globToRegex(glob: string): string {
  let out = "";
  for (const ch of glob) {
    switch (ch) {
      case "*":
        out += ".*";
        break;
      case "?":
        out += ".";
        break;
      case ".":
      case "+":
      case "(":
      case ")":
      case "|":
      case "^":
      case "$":
      case "[":
      case "]":
      case "{":
      case "}":
      case "\\":
        out += `\\${ch}`;
        break;
      default:
        out += ch;
    }
  }
  return out;
}

function compileIgnorePattern(pattern: string): RegExp | null {
  try {
    let src: string;
    if (/[*?]/.test(pattern)) {
      src = globToRegex(pattern);
    } else {
      try {
        new RegExp(pattern);
        src = pattern;
      } catch {
        src = globToRegex(pattern);
      }
    }
    return new RegExp(src, "i");
  } catch {
    return null;
  }
}

export function validateIgnorePatterns(patterns: string[]): string | null {
  for (const p of patterns) {
    if (!compileIgnorePattern(p)) {
      return `忽略规则无效：${p}（支持正则或通配符 * ?）`;
    }
  }
  return null;
}
