import Graph from 'graphology';
import { describe, expect, it } from 'vitest';
import { pathEdges, shortestPath } from './graphAlgo';

function sample() {
  const g = new Graph({ type: 'directed' });
  ['a', 'b', 'c', 'd'].forEach((n) => g.addNode(n));
  g.addDirectedEdgeWithKey('a>b', 'a', 'b', { count: 1 });
  g.addDirectedEdgeWithKey('b>c', 'b', 'c', { count: 1 });
  return g;
}

describe('shortestPath', () => {
  it('finds directed and undirected paths', () => {
    const g = sample();
    expect(shortestPath(g, 'a', 'c', true)).toEqual(['a', 'b', 'c']);
    expect(shortestPath(g, 'c', 'a', true)).toBeNull();
    expect(shortestPath(g, 'c', 'a', false)).toEqual(['c', 'b', 'a']);
  });
  it('handles identity, isolated and unknown nodes', () => {
    const g = sample();
    expect(shortestPath(g, 'a', 'a', true)).toEqual(['a']);
    expect(shortestPath(g, 'a', 'd', false)).toBeNull();
    expect(shortestPath(g, 'a', 'zzz', false)).toBeNull();
  });
  it('lists traversed edges in either direction', () => {
    const g = sample();
    expect(pathEdges(g, ['c', 'b', 'a'])).toEqual(['b>c', 'a>b']);
  });
});
