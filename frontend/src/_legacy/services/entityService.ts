import { Entity, CreateEntityInput, EntityAddress } from '../types/entity';
import { NetworkID } from '../types/blockchain';

const LOCAL_STORAGE_KEY = 'xolograph_entities_db';

class EntityService {
  private getStorage(): Entity[] {
    const raw = localStorage.getItem(LOCAL_STORAGE_KEY);
    if (!raw) {
      const seed: Entity = {
        id: 'ent_demo_01', investigationId: 'inv_demo_01', name: 'Tornado Cash Intermediary',
        category: 'MIXER_NODE', attributionSource: 'ANALYST_ASSERTION', confidence: 'MEDIUM',
        supportingEvidence: 'Pattern matching high-velocity incoming transfers.',
        analystNotes: 'Requires further behavioral corroboration.',
        createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
        addresses: [{ entityId: 'ent_demo_01', address: '0x71c7656ec7ab88b098defb751b7401b5f6d8976f', network: 'ethereum-mainnet', dateAttached: new Date().toISOString(), attachedBy: 'Det. Lead' }]
      };
      localStorage.setItem(LOCAL_STORAGE_KEY, JSON.stringify([seed]));
      return [seed];
    }
    return JSON.parse(raw);
  }

  private setStorage(data: Entity[]): void { localStorage.setItem(LOCAL_STORAGE_KEY, JSON.stringify(data)); }

  async listEntities(investigationId: string): Promise<Entity[]> {
    return this.getStorage().filter((e) => e.investigationId === investigationId);
  }

  async createEntity(input: CreateEntityInput): Promise<Entity> {
    const all = this.getStorage();
    const newEntity: Entity = { id: `ent_${Date.now()}`, ...input, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(), addresses: [] };
    all.unshift(newEntity);
    this.setStorage(all);
    return newEntity;
  }
}

export const entityService = new EntityService();