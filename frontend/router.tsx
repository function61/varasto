import { makeRouter } from 'f61ui/typescript-safe-router/saferouter';
import BrowsePage from 'pages/BrowsePage';
import ClientsPage from 'pages/ClientsPage';
import CollectionPage from 'pages/CollectionPage';
import EncryptionKeysPage from 'pages/EncryptionKeysPage';
import HealthPage from 'pages/HealthPage';
import NodesPage from 'pages/NodesPage';
import ReplicationPoliciesPage from 'pages/ReplicationPoliciesPage';
import ServerInfoPage from 'pages/ServerInfoPage';
import UsersPage from 'pages/UsersPage';
import VolumesAndMountsPage from 'pages/VolumesAndMountsPage';
import * as React from 'react';
import {
	browseRoute,
	clientsRoute,
	collectionRoute,
	encryptionKeysRoute,
	healthRoute,
	nodesRoute,
	replicationPoliciesRoute,
	serverInfoRoute,
	usersRoute,
	volumesAndMountsRoute,
} from 'routes';

export const router = makeRouter(browseRoute, (opts) => (
	<BrowsePage directoryId={opts.dir} key={opts.dir} />
))
	.registerRoute(clientsRoute, () => <ClientsPage />)
	.registerRoute(nodesRoute, () => <NodesPage />)
	.registerRoute(serverInfoRoute, () => <ServerInfoPage />)
	.registerRoute(volumesAndMountsRoute, () => <VolumesAndMountsPage />)
	.registerRoute(healthRoute, () => <HealthPage />)
	.registerRoute(usersRoute, () => <UsersPage />)
	.registerRoute(encryptionKeysRoute, () => <EncryptionKeysPage />)
	.registerRoute(collectionRoute, (opts) => (
		<CollectionPage
			id={opts.id}
			rev={opts.rev}
			pathBase64={opts.path}
			key={`${opts.id}/${opts.rev}/${opts.path}`}
		/>
	))
	.registerRoute(replicationPoliciesRoute, () => <ReplicationPoliciesPage />);
