import { useDroppable } from "@dnd-kit/core";
import { DndContext, type DragEndEvent, PointerSensor, useSensor, useSensors } from "@dnd-kit/core";
import type { Application, Status } from "../types";
import { STATUSES, STATUS_LABELS } from "../types";
import { ApplicationCard } from "./ApplicationCard";

interface Props {
  applications: Application[];
  onMove: (id: number, status: Status) => void;
  onOpen: (id: number) => void;
}

function Column({
  status,
  apps,
  onOpen,
}: {
  status: Status;
  apps: Application[];
  onOpen: (id: number) => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: status });
  return (
    <div
      ref={setNodeRef}
      className={`column ${isOver ? "column-over" : ""}`}
    >
      <div className="column-header">
        {STATUS_LABELS[status]} <span className="count">{apps.length}</span>
      </div>
      <div className="column-body">
        {apps.map((a) => (
          <ApplicationCard key={a.id} app={a} onOpen={onOpen} />
        ))}
      </div>
    </div>
  );
}

export function KanbanBoard({ applications, onMove, onOpen }: Props) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } })
  );

  function handleDragEnd(e: DragEndEvent) {
    const id = Number(e.active.id);
    const target = e.over?.id as Status | undefined;
    if (!target) return;
    const app = applications.find((a) => a.id === id);
    if (app && app.current_status !== target) {
      onMove(id, target);
    }
  }

  return (
    <DndContext sensors={sensors} onDragEnd={handleDragEnd}>
      <div className="board">
        {STATUSES.map((s) => (
          <Column
            key={s}
            status={s}
            apps={applications.filter((a) => a.current_status === s)}
            onOpen={onOpen}
          />
        ))}
      </div>
    </DndContext>
  );
}
