// SPDX-License-Identifier: AGPL-3.0-or-later

import mark from "../../assets/logo-mark.svg";

// The craftsail growth mark: two sails of rising height over a hull that
// climbs to the right. Source: web/src/assets/logo-mark.svg.
export function LogoMark({ size = 36, className = "" }: { size?: number; className?: string }) {
  return <img src={mark} width={size} height={size} alt="" className={"shrink-0 " + className} />;
}

export function Wordmark({ className = "" }: { className?: string }) {
  return (
    <span className={"truncate text-lg font-bold tracking-tight text-gray-900 " + className}>
      craftsail<span className="font-semibold text-primary-600"> growth</span>
    </span>
  );
}

export function Logo({ size = 36 }: { size?: number }) {
  return (
    <span className="flex min-w-0 items-center gap-2.5">
      <LogoMark size={size} />
      <Wordmark />
    </span>
  );
}
