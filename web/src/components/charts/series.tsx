// SPDX-License-Identifier: AGPL-3.0-or-later

import { useEffect, useMemo, useRef } from "react";
import { BarChart, LineChart, PieChart } from "echarts/charts";
import { GridComponent, LegendComponent, MarkLineComponent, MarkPointComponent, TooltipComponent, DataZoomComponent } from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { useI18n } from "../../i18n";

echarts.use([
  LineChart,
  BarChart,
  PieChart,
  GridComponent,
  LegendComponent,
  TooltipComponent,
  MarkLineComponent,
  MarkPointComponent,
  DataZoomComponent,
  CanvasRenderer,
]);

export type ChartPoint = { day: string; value: number | null; failed?: number };
type ChartTheme = "light" | "dark";

const FAIL = "#f1511c";

type LineProps = {
  label: string;
  points: ChartPoint[];
  height?: number;
  color?: string;
  reversed?: boolean;
  empty?: string;
  format?: (value: number) => string;
  target?: number;
  caption?: string;
  theme?: ChartTheme;
};

function knownOf(points: ChartPoint[]) {
  return points.filter((point) => point.value != null);
}

function palette(theme: ChartTheme) {
  if (theme === "dark") {
    return {
      text: "#a09d95",
      line: "#343333",
      split: "#2a2724",
      accent: "#f1511c",
      area: "rgba(241,81,28,0.18)",
      tip: "#191919",
    };
  }
  return {
    text: "#64748b",
    line: "#e7eeec",
    split: "#eef2f1",
    accent: "#0d9488",
    area: "rgba(13,148,136,0.16)",
    tip: "#fff",
  };
}

function useChart(option: echarts.EChartsCoreOption | null) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const node = ref.current;
    if (!node || !option) return;
    const inst = echarts.init(node, undefined, { renderer: "canvas" });
    inst.setOption(option, true);
    const ro = new ResizeObserver(() => inst.resize());
    ro.observe(node);
    return () => {
      ro.disconnect();
      inst.dispose();
    };
  }, [option]);
  return ref;
}

export function TimeSeries({
  label,
  points,
  height = 260,
  color,
  reversed = false,
  empty,
  format = (value) => String(Math.round(value * 100) / 100),
  target,
  caption,
  theme = "light",
}: LineProps) {
  const { t } = useI18n();
  const emptyText = empty ?? t("charts.noLine");
  const known = knownOf(points);
  const option = useMemo(() => {
    if (known.length < 2) return null;
    const colors = palette(theme);
    const stroke = color || colors.accent;
    const days = points.map((point) => point.day.slice(5) || point.day);
    const fails = points.flatMap((point, index) =>
      point.failed && point.value != null
        ? [{ name: t("charts.failed"), coord: [days[index], point.value] as [string, number] }]
        : [],
    );
    return {
      animationDuration: 480,
      tooltip: {
        trigger: "axis",
        backgroundColor: colors.tip,
        borderColor: colors.line,
        textStyle: { color: theme === "dark" ? "#ece7db" : "#0f172a", fontSize: 12 },
        valueFormatter: (value: unknown) => (typeof value === "number" ? format(value) : t("charts.noData")),
      },
      grid: { left: 52, right: 18, top: 18, bottom: points.length > 18 ? 52 : 28 },
      dataZoom: points.length > 18 ? [{ type: "inside" }, { type: "slider", height: 16, bottom: 8 }] : undefined,
      xAxis: {
        type: "category",
        data: days,
        boundaryGap: false,
        axisLine: { lineStyle: { color: colors.line } },
        axisLabel: { color: colors.text, fontSize: 11 },
      },
      yAxis: {
        type: "value",
        inverse: reversed,
        scale: true,
        axisLabel: { color: colors.text, fontSize: 11, formatter: (value: number) => format(value) },
        splitLine: { lineStyle: { color: colors.split } },
      },
      series: [
        {
          name: label,
          type: "line",
          data: points.map((point) => point.value),
          smooth: 0.25,
          connectNulls: false,
          showSymbol: true,
          symbolSize: 7,
          lineStyle: { width: 2.5, color: stroke },
          itemStyle: { color: stroke },
          areaStyle: { color: colors.area },
          markLine: target == null ? undefined : {
            symbol: "none",
            label: { formatter: () => `Target ${format(target)}`, color: colors.text },
            lineStyle: { type: "dashed", color: "#94a3b8" },
            data: [{ yAxis: target }],
          },
          markPoint: fails.length
            ? { symbol: "circle", symbolSize: 12, itemStyle: { color: FAIL }, data: fails }
            : undefined,
        },
      ],
    } as echarts.EChartsCoreOption;
  }, [points, color, reversed, format, target, label, known.length, theme]);
  const ref = useChart(option);
  if (!option) {
    return (
      <div className="card p-5">
        <div className="stat-label">{label}</div>
        <p className="hint">{emptyText}</p>
        {caption && <p className="hint">{caption}</p>}
      </div>
    );
  }
  return (
    <div className="card p-5">
      <div className="toolbar" style={{ marginBottom: 4 }}>
        <div className="stat-label">{label}</div>
        <span className="hint">{known[known.length - 1].day} · {format(known[known.length - 1].value as number)}</span>
      </div>
      {caption && <p className="hint">{caption}</p>}
      <div ref={ref} style={{ width: "100%", height }} />
      {points.some((point) => point.failed) && <p className="hint">{t("charts.failedNote")}</p>}
    </div>
  );
}

function pctAxis(value: number) {
  return `${Math.round(value)}%`;
}

export function AreaSeries({
  label,
  points,
  color,
  height = 88,
  empty,
}: {
  label: string;
  points: ChartPoint[];
  color: string;
  height?: number;
  empty?: string;
}) {
  const { t } = useI18n();
  const emptyText = empty ?? t("charts.noDays");
  const option = useMemo(() => {
    if (knownOf(points).length < 2) return null;
    return {
      animationDuration: 400,
      tooltip: { trigger: "axis", valueFormatter: (value: unknown) => (typeof value === "number" ? pctAxis(value) : "—") },
      grid: { left: 42, right: 12, top: 12, bottom: 24 },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: points.map((point) => point.day.slice(5) || point.day),
        axisLine: { lineStyle: { color: "#e7eeec" } },
        axisLabel: { color: "#64748b", fontSize: 11 },
      },
      yAxis: {
        type: "value",
        axisLabel: { color: "#64748b", fontSize: 11, formatter: pctAxis },
        splitLine: { lineStyle: { color: "#eef2f1" } },
      },
      series: [{
        name: label,
        type: "line",
        smooth: 0.3,
        showSymbol: points.length < 20,
        data: points.map((point) => point.value),
        lineStyle: { width: 2, color },
        itemStyle: { color },
        areaStyle: { color: "rgba(37,99,235,0.12)" },
      }],
    } as echarts.EChartsCoreOption;
  }, [points, label, color]);
  const ref = useChart(option);
  if (!option) return <p className="hint">{emptyText}</p>;
  return <div ref={ref} style={{ width: "100%", height }} />;
}

export function LineSeries({
  rows,
  series,
  height = 260,
  empty,
}: {
  rows: Array<Record<string, string | number | null>>;
  series: { key: string; name: string; color: string }[];
  height?: number;
  empty?: string;
}) {
  const { t } = useI18n();
  const emptyText = empty ?? t("charts.noPoints");
  const option = useMemo(() => {
    if (rows.length < 1 || series.length < 1) return null;
    return {
      animationDuration: 400,
      tooltip: { trigger: "axis", valueFormatter: (value: unknown) => (typeof value === "number" ? pctAxis(value) : "—") },
      legend: { bottom: 0, textStyle: { color: "#64748b", fontSize: 12 } },
      grid: { left: 42, right: 12, top: 16, bottom: 42 },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: rows.map((row) => String(row.day || "").slice(5)),
        axisLabel: { color: "#64748b", fontSize: 11 },
      },
      yAxis: {
        type: "value",
        min: 0,
        max: 100,
        axisLabel: { color: "#64748b", fontSize: 11, formatter: "{value}%" },
        splitLine: { lineStyle: { color: "#eef2f1" } },
      },
      series: series.map((item) => ({
        name: item.name,
        type: "line",
        smooth: 0.2,
        showSymbol: rows.length < 24,
        data: rows.map((row) => (typeof row[item.key] === "number" ? row[item.key] : null)),
        lineStyle: { width: item.key === "brand" ? 3 : 2, color: item.color },
        itemStyle: { color: item.color },
      })),
    } as echarts.EChartsCoreOption;
  }, [rows, series]);
  const ref = useChart(option);
  if (!option) return <p className="hint">{emptyText}</p>;
  return <div ref={ref} style={{ width: "100%", height }} />;
}

export function DonutSeries({
  slices,
  height = 180,
}: {
  slices: { name: string; value: number; color: string; tip?: string }[];
  height?: number;
}) {
  const option = useMemo(() => {
    if (!slices.length) return null;
    return {
      tooltip: {
        trigger: "item",
        formatter: (item: { name?: string; data?: { tip?: string } }) => item.data?.tip || item.name || "",
      },
      series: [{
        type: "pie",
        radius: ["52%", "78%"],
        label: { show: false },
        data: slices.map((slice) => ({ name: slice.name, value: slice.value, tip: slice.tip, itemStyle: { color: slice.color } })),
      }],
    } as echarts.EChartsCoreOption;
  }, [slices]);
  const ref = useChart(option);
  if (!option) return null;
  return <div ref={ref} style={{ width: "100%", height }} />;
}

export function StackSeries({
  rows,
  keys,
  meta,
  height = 220,
  empty,
}: {
  rows: Array<Record<string, number | string>>;
  keys: string[];
  meta: Record<string, { label: string; color: string }>;
  height?: number;
  empty?: string;
}) {
  const { t } = useI18n();
  const emptyText = empty ?? t("charts.noCitations");
  const option = useMemo(() => {
    if (rows.length < 1) return null;
    return {
      tooltip: { trigger: "axis", valueFormatter: (value: unknown) => (typeof value === "number" ? pctAxis(value) : "—") },
      legend: { bottom: 0, textStyle: { fontSize: 11, color: "#64748b" } },
      grid: { left: 42, right: 12, top: 12, bottom: 42 },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: rows.map((row) => String(row.day || "").slice(5)),
        axisLabel: { color: "#64748b", fontSize: 11 },
      },
      yAxis: {
        type: "value",
        min: 0,
        max: 100,
        axisLabel: { formatter: "{value}%", color: "#64748b", fontSize: 11 },
        splitLine: { lineStyle: { color: "#eef2f1" } },
      },
      series: keys.map((key) => ({
        name: meta[key]?.label || key,
        type: "line",
        stack: "share",
        areaStyle: {},
        showSymbol: false,
        data: rows.map((row) => (typeof row[key] === "number" ? row[key] : 0)),
        itemStyle: { color: meta[key]?.color || "#94a3b8" },
        lineStyle: { width: 1, color: meta[key]?.color || "#94a3b8" },
      })),
    } as echarts.EChartsCoreOption;
  }, [rows, keys, meta]);
  const ref = useChart(option);
  if (!option) return <p className="hint">{emptyText}</p>;
  return <div ref={ref} style={{ width: "100%", height }} />;
}
