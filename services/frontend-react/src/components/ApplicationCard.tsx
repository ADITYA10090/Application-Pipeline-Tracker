import { useDraggable } from "@dnd-kit/core";
import type { Application } from "../types";

interface Props {
  app: Application;
  onOpen: (id: number) => void;
}

export function ApplicationCard({ app, onOpen }: Props) {
  const { attributes, listeners, setNodeRef, transform, isDragging } =
    useDraggable({ id: app.id });

  const style: React.CSSProperties = {
    transform: transform
      ? `translate3d(${transform.x}px, ${transform.y}px, 0)`
      : undefined,
    opacity: isDragging ? 0.4 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="card"
      {...listeners}
      {...attributes}
    >
      <div className="card-title" onClick={() => onOpen(app.id)}>
        {app.title}
      </div>
      <div className="card-company">{app.company}</div>
      {app.match_score != null && (
        <div className="card-score">
          match {(app.match_score * 100).toFixed(0)}%
        </div>
      )}
    </div>
  );
}
