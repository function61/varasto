import { makeRoute } from 'f61ui/typescript-safe-router/saferouter';

export const browseRoute = makeRoute('browse', { dir: 'string' });

export const clientsRoute = makeRoute('clients', {});

export const nodesRoute = makeRoute('nodes', {});

export const serverInfoRoute = makeRoute('serverInfo', {});

export const volumesAndMountsRoute = makeRoute('volumesmounts', {});

export const replicationPoliciesRoute = makeRoute('replicationpolicies', {});

export const collectionRoute = makeRoute('collection', {
	id: 'string',
	rev: 'string',
	path: 'string',
});
