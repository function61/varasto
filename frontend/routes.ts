import { makeRoute } from 'f61ui/typescript-safe-router/saferouter';

export const browseRoute = makeRoute('browse', { dir: 'string', v: 'string' });

export const nodesRoute = makeRoute('nodes', {});

export const metadataBackupRoute = makeRoute('metadataBackup', { v: 'string' });

export const usersRoute = makeRoute('users', {});

export const serverInfoRoute = makeRoute('serverInfo', {});

export const volumesAndMountsRoute = makeRoute('volumesmounts', { view: 'string' });

export const replicationPoliciesRoute = makeRoute('replicationpolicies', {});

export const logsRoute = makeRoute('logs', {});

export const contentMetadataRoute = makeRoute('contentMetadata', {});

export const fuseServerRoute = makeRoute('fuseServer', {});

export const collectionRoute = makeRoute('collection', {
	id: 'string',
	rev: 'string',
	path: 'string',
});
