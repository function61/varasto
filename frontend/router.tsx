import { makeRouter } from 'f61ui/typescript-safe-router/saferouter';
import BrowsePage from 'pages/BrowsePage';
import CollectionPage from 'pages/CollectionPage';
import ContentMetadataPage from 'pages/ContentMetadataPage';
import FuseServerPage from 'pages/FuseServerPage';
import LogsPage from 'pages/LogsPage';
import MetadataBackupPage from 'pages/MetadataBackupPage';
import NodesPage from 'pages/NodesPage';
import ReplicationPoliciesPage from 'pages/ReplicationPoliciesPage';
import ServerInfoPage from 'pages/ServerInfoPage';
import UsersPage from 'pages/UsersPage';
import VolumesAndMountsPage from 'pages/VolumesAndMountsPage';
import * as React from 'react';
import * as r from 'routes';

export const router = makeRouter(r.browseRoute, (opts) => (
	<BrowsePage directoryId={opts.dir} key={opts.dir} />
))
	.registerRoute(r.nodesRoute, () => <NodesPage />)
	.registerRoute(r.fuseServerRoute, () => <FuseServerPage />)
	.registerRoute(r.serverInfoRoute, () => <ServerInfoPage />)
	.registerRoute(r.volumesAndMountsRoute, () => <VolumesAndMountsPage />)
	.registerRoute(r.usersRoute, () => <UsersPage />)
	.registerRoute(r.logsRoute, () => <LogsPage />)
	.registerRoute(r.metadataBackupRoute, () => <MetadataBackupPage />)
	.registerRoute(r.contentMetadataRoute, () => <ContentMetadataPage />)
	.registerRoute(r.collectionRoute, (opts) => (
		<CollectionPage
			id={opts.id}
			rev={opts.rev}
			pathBase64={opts.path}
			key={`${opts.id}/${opts.rev}/${opts.path}`}
		/>
	))
	.registerRoute(r.replicationPoliciesRoute, () => <ReplicationPoliciesPage />);
