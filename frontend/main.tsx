import { boot, makeRouter } from 'f61ui/appcontroller';
import { DangerAlert } from 'f61ui/component/alerts';
import { navigateTo } from 'f61ui/browserutils';
import { GlobalConfig } from 'f61ui/globalconfig';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import BrowsePage from 'pages/BrowsePage';
import CollectionPage from 'pages/CollectionPage';
import ContentMetadataPage from 'pages/ContentMetadataPage';
import DownloadClientAppPage from 'pages/DownloadClientAppPage';
import { RootFolderId } from 'generated/stoserver/stoservertypes_types';
import FuseServerPage from 'pages/FuseServerPage';
import GettingStartedPage from 'pages/GettingStartedPage';
import LogsPage from 'pages/LogsPage';
import MetadataBackupPage from 'pages/MetadataBackupPage';
import NodesPage from 'pages/NodesPage';
import MetricsPage from 'pages/MetricsPage';
import ReplicationPoliciesPage from 'pages/ReplicationPoliciesPage';
import ScheduledJobsPage from 'pages/ScheduledJobsPage';
import ServerInfoPage from 'pages/ServerInfoPage';
import SubsystemsPage from 'pages/SubsystemsPage';
import UsersPage from 'pages/UsersPage';
import VolumesAndMountsPage from 'pages/VolumesAndMountsPage';
import * as React from 'react';
import * as r from 'generated/frontend_uiroutes';

class Handlers implements r.RouteHandlers {
	root() {
		navigateTo(r.browseUrl({ dir: RootFolderId, view: '' }));
		return <h1>Redirect</h1>;
	}

	browse(opts: r.BrowseOpts) {
		return <BrowsePage key={opts.dir} directoryId={opts.dir} view={opts.view} />;
	}

	collection(opts: r.CollectionOpts) {
		return (
			<CollectionPage
				key={`${opts.id}/${opts.rev}/${opts.path}`}
				id={opts.id}
				rev={opts.rev}
				page={opts.page}
				view={opts.view}
				pathBase64={opts.path}
			/>
		);
	}

	gettingStarted(opts: r.GettingStartedOpts) {
		return <GettingStartedPage view={opts.section} />;
	}

	downloadClientApp() {
		return <DownloadClientAppPage />;
	}

	nodes() {
		return <NodesPage />;
	}

	metadataBackup(opts: r.MetadataBackupOpts) {
		return <MetadataBackupPage view={opts.view} />;
	}

	users() {
		return <UsersPage />;
	}

	serverInfo() {
		return <ServerInfoPage />;
	}

	subsystems() {
		return <SubsystemsPage />;
	}

	volumes() {
		return <VolumesAndMountsPage view="volumes" />;
	}

	mounts() {
		return <VolumesAndMountsPage view="mounts" />;
	}

	volumesTopologyZones() {
		return <VolumesAndMountsPage view="topology" />;
	}

	volumesService() {
		return <VolumesAndMountsPage view="service" />;
	}

	volumesSmart() {
		return <VolumesAndMountsPage view="smart" />;
	}

	volumesIntegrity() {
		return <VolumesAndMountsPage view="integrity" />;
	}

	volumesReplication() {
		return <VolumesAndMountsPage view="replicationStatuses" />;
	}

	replicationPolicies() {
		return <ReplicationPoliciesPage />;
	}

	logs() {
		return <LogsPage />;
	}

	metrics() {
		return <MetricsPage />;
	}

	contentMetadata() {
		return <ContentMetadataPage />;
	}

	fuseServer() {
		return <FuseServerPage />;
	}

	scheduledJobs() {
		return <ScheduledJobsPage />;
	}

	notFound(url: string) {
		return (
			<AppDefaultLayout title="404" breadcrumbs={[]}>
				<h1>404</h1>

				<DangerAlert>The page ({url}) you were looking for, is not found.</DangerAlert>
			</AppDefaultLayout>
		);
	}
}

export function main(appElement: HTMLElement, config: GlobalConfig): void {
	const handlers = new Handlers();

	boot(
		appElement,
		config,
		makeRouter(r.hasRouteFor, (relativeUrl: string) => r.handle(relativeUrl, handlers)),
	);
}
