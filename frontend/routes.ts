import { makeRoute } from 'f61ui/typescript-safe-router/saferouter';

// would've liked to keep 2nd param "view" as "v" for short, but have to work around this issue:
// https://github.com/FrigoEU/typescript-safe-router/issues/4
export const browseRoute = makeRoute('browse', { dir: 'string', view: 'string' });

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
