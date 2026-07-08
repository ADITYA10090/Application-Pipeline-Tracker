import { useMemo } from "react";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { useState } from "react";
import type { Application, Status } from "../types";
import { STATUSES, STATUS_LABELS } from "../types";

interface Props {
  applications: Application[];
  onMove: (id: number, status: Status) => void;
  onOpen: (app: Application) => void;
}

function Card({
  app,
  onOpen,
}: {
  app: Application;
  onOpen: (a: Application) => void;
}) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: app.id,
  });
  return (
    <div
      ref={setNodeRef}
      className="card"
      style={{ opacity: isDragging ? 0.4 : 1 }}
      {...listeners}
      {...attributes}
      onClick={() => onOpen(app)}
    >
      <div className="title">{app.title}</div>
      <div className="company">
        {app.company}
        {app.location ? ` · ${app.location}` : ""}
      </div>
      {app.match_score != null && (
        <span className="score">{Math.round(app.match_score * 100)}% match</span>
      )}
    </div>
  );
}

function Column({
  status,
  apps,
  onOpen,
}: {
  status: Status;
  apps: Application[];
  onOpen: (a: Application) => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: status });
  return (
    <div ref={setNodeRef} className={`column ${isOver ? "drop-over" : ""}`}>
      <div className="column-header">
        <span>{STATUS_LABELS[status]}</span>
        <span className="column-count">{apps.length}</span>
      </div>
      <div className="column-body">
        {apps.map((a) => (
          <Card key={a.id} app={a} onOpen={onOpen} />
        ))}
      </div>
    </div>
  );
}

export function KanbanBoard({ applications, onMove, onOpen }: Props) {
  const [activeId, setActiveId] = useState<number | null>(null);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  const byStatus = useMemo(() => {
    const map: Record<Status, Application[]> = {
      wishlist: [],
      applied: [],
      oa: [],
      interview: [],
      offer: [],
      rejected: [],
    };
    for (const a of applications) map[a.current_status].push(a);
    return map;
  }, [applications]);

  const activeApp = applications.find((a) => a.id === activeId) ?? null;

  function handleStart(e: DragStartEvent) {
    setActiveId(Number(e.active.id));
  }

  function handleEnd(e: DragEndEvent) {
    setActiveId(null);
    const overId = e.over?.id as Status | undefined;
    if (!overId) return;
    const app = applications.find((a) => a.id === Number(e.active.id));
    if (app && app.current_status !== overId) onMove(app.id, overId);
  }

  return (
    <DndContext sensors={sensors} onDragStart={handleStart} onDragEnd={handleEnd}>
      <div className="board">
        {STATUSES.map((s) => (
          <Column key={s} status={s} apps={byStatus[s]} onOpen={onOpen} />
        ))}
      </div>
      <DragOverlay>
        {activeApp ? (
          <div className="card">
            <div className="title">{activeApp.title}</div>
            <div className="company">{activeApp.company}</div>
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}
