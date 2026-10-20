"use client";

import * as React from "react";
import { DynamicIcon, iconNames, type IconName } from "lucide-react/dynamic";

const lucideIconNames = new Set<string>(iconNames);

/**
 * Renders a tool's `icon` value, which is either a Lucide icon name
 * (e.g. "heart-pulse") or an emoji. Anything else renders the fallback so
 * raw identifiers never show up as text.
 */
export function ToolIcon({
  icon,
  fallback,
}: {
  icon?: string;
  fallback: React.ReactNode;
}) {
  const value = icon?.trim() ?? "";
  if (lucideIconNames.has(value)) {
    return (
      <DynamicIcon
        name={value as IconName}
        className="h-4 w-4 shrink-0"
        fallback={() => <>{fallback}</>}
      />
    );
  }
  if (/\p{Extended_Pictographic}/u.test(value)) {
    return <span className="text-lg leading-none">{value}</span>;
  }
  return <>{fallback}</>;
}
