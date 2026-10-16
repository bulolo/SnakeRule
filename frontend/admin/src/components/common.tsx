import { useEffect, useRef, useState } from "react";
import { snakeRuleApi } from "../lib/sdk/generated";

export function CustomSelect({
  value,
  options,
  onChange,
  disabled = false,
  ariaLabel,
}: {
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
  disabled?: boolean;
  ariaLabel: string;
}) {
  const root = useRef<HTMLDetailsElement>(null);
  const selected = options.find((option) => option.value === value);
  useEffect(() => {
    function closeOnOutsideClick(event: PointerEvent) {
      if (!root.current?.contains(event.target as Node)) {
        if (root.current) root.current.open = false;
      }
    }
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape" && root.current?.open) {
        root.current.open = false;
        root.current.querySelector("summary")?.focus();
      }
    }
    document.addEventListener("pointerdown", closeOnOutsideClick);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutsideClick);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, []);
  return (
    <details className="custom-select" ref={root}>
      <summary
        aria-label={ariaLabel}
        aria-disabled={disabled}
        onClick={(event) => {
          if (disabled) event.preventDefault();
        }}
        className={
          disabled ? "custom-select-trigger disabled" : "custom-select-trigger"
        }
      >
        {selected?.label ?? value}
      </summary>
      {!disabled && (
        <div
          className="custom-select-menu"
          role="listbox"
          aria-label={ariaLabel}
        >
          {options.map((option) => (
            <button
              type="button"
              role="option"
              aria-selected={option.value === value}
              key={option.value}
              onClick={() => {
                onChange(option.value);
                if (root.current) root.current.open = false;
              }}
            >
              {option.label}
            </button>
          ))}
        </div>
      )}
    </details>
  );
}

export function LoginScreen({
  onSuccess,
  brand,
}: {
  onSuccess: (email: string) => void;
  brand?: { name: string; tagline: string; logo_url: string };
}) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const session = await snakeRuleApi.login({
        email: email.trim(),
        password,
      });
      onSuccess(session.user?.email ?? email.trim());
    } catch (e) {
      setError(e instanceof Error ? e.message : "登录失败，请重试。");
    } finally {
      setBusy(false);
    }
  }
  return (
    <main className="login-page">
      <div className="login-card">
        <a className="brand login-brand" href="/">
          <span className={`brand-icon${brand?.logo_url ? " has-logo" : ""}`}>{brand?.logo_url ? <img src={brand.logo_url} alt="" /> : (brand?.name?.[0] ?? "S")}</span>
          <span>
            {brand?.name ?? "SnakeRule"}<small>{brand?.tagline || "RULE ENGINE"}</small>
          </span>
        </a>
        <div className="login-heading">
          <small className="eyebrow">SECURE ORGANIZATION</small>
          <h1>欢迎回来</h1>
          <p>登录以管理规则集并运行决策测试。</p>
        </div>
        <form onSubmit={submit}>
          <label>
            邮箱
            <input
              type="email"
              autoComplete="email"
              autoFocus
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="输入管理员邮箱"
            />
          </label>
          <label>
            密码
            <input
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="输入密码"
            />
          </label>
          {error && (
            <div className="login-error" role="alert">
              {error}
            </div>
          )}
          <button className="primary login-submit" disabled={busy}>
            {busy ? "正在登录…" : "登录工作台"} <span>→</span>
          </button>
        </form>
        <div className="login-foot">
          <i /> 安全会话连接 <span>·</span> {brand?.name ?? "SnakeRule"}
        </div>
      </div>
      <small className="login-version">{brand?.tagline || "规则引擎管理控制台"}</small>
    </main>
  );
}
