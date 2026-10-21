import React, { useState, useEffect } from 'react';
import { Entity } from '../types/entity';
import { entityService } from '../services/entityService';

export const EntityIntelligenceView: React.FC = () => {
  const [entities, setEntities] = useState<Entity[]>([]);

  useEffect(() => {
    entityService.listEntities('inv_demo_01').then(setEntities);
  }, []);

  return (
    <div className="flex h-full flex-col space-y-4">
      <h1 className="text-xl font-bold uppercase tracking-wider text-[var(--text-primary)]">ENTITY INTELLIGENCE & CLUSTERING</h1>
      <div className="flex-1 overflow-y-auto">
        {entities.map((e) => (
          <div key={e.id} className="p-4 border-b border-[var(--border-subtle)] bg-[var(--bg-elevated)] mb-2">
            <span className="font-bold text-sm text-[var(--text-primary)]">{e.name}</span>
            <div className="text-xs text-[var(--text-muted)]">Source: {e.attributionSource}</div>
          </div>
        ))}
      </div>
    </div>
  );
};