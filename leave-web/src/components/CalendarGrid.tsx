import { memo, useMemo } from "react";
import type { CalendarDayCapacity, CalendarEvent } from "../types";
import { addDays, isoDate, isWeekend } from "../utils/date";

export interface CalendarGridProps {
  from: string; // inclusive YYYY-MM-DD
  days: number; // window size
  events: CalendarEvent[];
  capacities?: CalendarDayCapacity[];
  holidays?: string[];
  title?: string;
}

interface DayCell {
  date: string;
  isToday: boolean;
  isWeekend: boolean;
  isHoliday: boolean;
  capacityPct?: number;
  events: CalendarEvent[];
}

function buildCells(
  from: string,
  days: number,
  events: CalendarEvent[],
  capacities?: CalendarDayCapacity[],
  holidays?: string[]
): DayCell[] {
  const todayIso = isoDate(new Date());
  const capByDate = new Map<string, CalendarDayCapacity>();
  (capacities || []).forEach((c) => capByDate.set(c.date, c));
  const holidaySet = new Set(holidays || []);

  const cells: DayCell[] = [];
  const start = new Date(from + "T00:00:00");
  for (let i = 0; i < days; i++) {
    const d = addDays(start, i);
    const iso = isoDate(d);
    const evs = events.filter(
      (e) => iso >= e.startDate && iso <= e.endDate
    );
    const cap = capByDate.get(iso);
    cells.push({
      date: iso,
      isToday: iso === todayIso,
      isWeekend: isWeekend(d),
      isHoliday: holidaySet.has(iso),
      capacityPct: cap?.capacityPct,
      events: evs,
    });
  }
  return cells;
}

function CalendarGridInner({
  from,
  days,
  events,
  capacities,
  holidays,
  title,
}: CalendarGridProps) {
  const cells = useMemo(
    () => buildCells(from, days, events, capacities, holidays),
    [from, days, events, capacities, holidays]
  );

  return (
    <div className="calendar-wrap">
      {title && <div className="calendar-title">{title}</div>}
      <div className="calendar-legend">
        <span className="legend-swatch today" /> Today
        <span className="legend-swatch weekend" /> Weekend
        <span className="legend-swatch holiday" /> Holiday
        <span className="legend-swatch leave" /> On leave
      </div>
      <div className="calendar-grid">
        {cells.map((c) => {
          const classes = ["calendar-cell"];
          if (c.isToday) classes.push("today");
          if (c.isWeekend) classes.push("weekend");
          if (c.isHoliday) classes.push("holiday");
          if (c.events.length > 0) classes.push("has-leave");
          return (
            <div key={c.date} className={classes.join(" ")}>
              <div className="calendar-cell-date">
                {c.date.slice(5)}
                {typeof c.capacityPct === "number" && (
                  <span
                    className={`capacity ${
                      c.capacityPct >= 80
                        ? "high"
                        : c.capacityPct >= 50
                        ? "medium"
                        : "low"
                    }`}
                    title={`${c.capacityPct}% on leave`}
                  >
                    {c.capacityPct}%
                  </span>
                )}
              </div>
              <ul className="calendar-events">
                {c.events.slice(0, 3).map((e) => (
                  <li
                    key={`${e.id}-${c.date}`}
                    title={`${e.userName} · ${e.leaveTypeName}`}
                  >
                    <span className="dot" /> {e.userName}
                  </li>
                ))}
                {c.events.length > 3 && (
                  <li className="more">+{c.events.length - 3} more</li>
                )}
              </ul>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export const CalendarGrid = memo(CalendarGridInner);
