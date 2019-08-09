import { makeRoute } from 'f61ui/typescript-safe-router/saferouter';

export const browseRoute = makeRoute('browse', { dir: 'string', v: 'string' });

export const clientsRoute = makeRoute('clients', {});

export const nodesRoute = makeRoute('nodes', {});

export const usersRoute = makeRoute('users', {});

export const serverInfoRoute = makeRoute('serverInfo', {});

export const volumesAndMountsRoute = makeRoute('volumesmounts', {});

export const replicationPoliciesRoute = makeRoute('replicationpolicies', {});

export const healthRoute = makeRoute('health', {});

export const logsRoute = makeRoute('logs', {});

export const encryptionKeysRoute = makeRoute('encryptionKeys', {});

export const contentMetadataRoute = makeRoute('contentMetadata', {});

export const fuseServerRoute = makeRoute('fuseServer', {});

export const collectionRoute = makeRoute('collection', {
	id: 'string',
	rev: 'string',
	path: 'string',
});
