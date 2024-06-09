import { DocUrlLatest, DocLink } from 'component/doclink';
import { volumeAutocomplete } from 'component/autocompletes';
import { thousandSeparate } from 'component/numberformatter';
import IntegrityVerificationJobsView, {
	volumeTechnologyBadge,
} from 'pages/views/IntegrityVerificationJobsView';
import SmartView from 'pages/views/SmartView';
import TopologyAndZonesView, { onlineBadge } from 'pages/views/TopologyAndZonesView';
import { RefreshButton } from 'component/refreshbutton';
import { Result } from 'f61ui/component/result';
import { TabController } from 'component/tabcontroller';
import { reloadCurrentPage } from 'f61ui/browserutils';
import { Glyphicon, Panel, CollapsePanel, tableClassStripedHover } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandIcon, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Info } from 'f61ui/component/info';
import { ProgressBar } from 'f61ui/component/progressbar';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import { plainDateToDateTime, dateRFC3339 } from 'f61ui/types';
import { unrecognizedValue } from 'f61ui/utils';
import {
	VolumeChangeDescription,
	VolumeChangeNotes,
	VolumeChangeQuota,
	VolumeCreate,
	VolumeMarkDataLost,
	VolumeDecommission,
	VolumeMigrateData,
	VolumeRemoveQueuedReplications,
	VolumeMountGoogleDrive,
	VolumeMountLocal,
	VolumeMountS3,
	VolumeRename,
	VolumeSetManufacturingDate,
	VolumeSetSerialNumber,
	VolumeSetTechnology,
	VolumeSetWarrantyEndDate,
	VolumeUnmount,
} from 'generated/stoserver/stoservertypes_commands';
import {
	getIntegrityVerificationJobs,
	getNodes,
	getReplicationStatuses,
	getVolumeMounts,
	getVolumes,
	getDecommissionedVolumes,
} from 'generated/stoserver/stoservertypes_endpoints';
import {
	DocRef,
	IntegrityVerificationJob,
	Node,
	ReplicationStatus,
	Volume,
	VolumeMount,
	VolumeTechnology,
} from 'generated/stoserver/stoservertypes_types';
import { AdminLayout } from 'layout/AdminLayout';
import * as React from 'react';
import {
	volumesUrl,
	mountsUrl,
	volumesTopologyZonesUrl,
	volumesServiceUrl,
	volumesSmartUrl,
	volumesIntegrityUrl,
	volumesReplicationUrl,
} from 'generated/frontend_uiroutes';

interface VolumesAndMountsPageProps {
	view:
		| 'volumes'
		| 'mounts'
		| 'topology'
		| 'service'
		| 'smart'
		| 'integrity'
		| 'replicationStatuses';
}

interface VolumesAndMountsPageState {
	volumes: Result<Volume[]>; // does not contain decommissioned ones
	volumesDecommissioned: Result<Volume[]>;
	mounts: Result<VolumeMount[]>;
	ivJobs: Result<IntegrityVerificationJob[]>;
	nodes: Result<Node[]>;
	replicationStatuses: Result<ReplicationStatus[]>;
}

export default class VolumesAndMountsPage extends React.Component<
	VolumesAndMountsPageProps,
	VolumesAndMountsPageState
> {
	state: VolumesAndMountsPageState = {
		volumes: new Result<Volume[]>((_) => {
			this.setState({ volumes: _ });
		}),
		volumesDecommissioned: new Result<Volume[]>((_) => {
			this.setState({ volumesDecommissioned: _ });
		}),
		mounts: new Result<VolumeMount[]>((_) => {
			this.setState({ mounts: _ });
		}),
		ivJobs: new Result<IntegrityVerificationJob[]>((_) => {
			this.setState({ ivJobs: _ });
		}),
		nodes: new Result<Node[]>((_) => {
			this.setState({ nodes: _ });
		}),
		replicationStatuses: new Result<ReplicationStatus[]>((_) => {
			this.setState({ replicationStatuses: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		const volumesHeading = () => (
			<div>
				Volumes &nbsp;
				<Dropdown>
					<CommandLink command={VolumeCreate()} />
				</Dropdown>
			</div>
		);

		const content = ((): React.ReactNode => {
			switch (this.props.view) {
				case 'integrity':
					return (
						<div>
							<Panel
								heading={
									<div>
										Data integrity verification jobs &nbsp;
										<Info text="Showing only latest scan result by default" />
										&nbsp;
										<DocLink
											doc={
												DocRef.DocsUsingBackgroundIntegrityVerificationIndexMd
											}
										/>
									</div>
								}>
								{this.renderIvJobs()}
							</Panel>
						</div>
					);
				case 'topology':
					return this.renderTopologyView();
				case 'service':
					return (
						<Panel
							heading={
								<div>
									Service view{' '}
									<Info text="If you have problems with a disk, find out its age, warranty details, serial number etc." />
								</div>
							}>
							{this.renderServiceView()}
						</Panel>
					);
				case 'replicationStatuses':
					return (
						<Panel
							heading={
								<div>
									Replication queue &nbsp;
									<Info text="Queued replication operations per volume. In a healthy system progress should be always realtime or close, unless you are doing large transfers or data migrations." />
									&nbsp;
									<RefreshButton
										refresh={() => {
											this.loadReplicationStatuses();
										}}
									/>
								</div>
							}>
							{this.renderReplicationStatuses()}
						</Panel>
					);
				case 'smart':
					return (
						<Panel
							heading={
								<div>
									SMART <DocLink doc={DocRef.DocsUsingSmartMonitoringIndexMd} />
								</div>
							}>
							{this.renderSmartView()}
						</Panel>
					);
				case 'volumes':
					return (
						<Panel heading={volumesHeading()}>
							{this.renderVolumes()}

							{this.renderDecommissionedVolumes()}
						</Panel>
					);
				case 'mounts':
					return (
						<Panel
							heading={
								<div>
									Mounts &nbsp;
									<Info text="Currently, mounting/unmounting makes Varasto automatically restart. So the UI giving out errors for a few seconds is to be expected." />
								</div>
							}>
							{this.renderMounts()}
						</Panel>
					);
				default:
					throw unrecognizedValue(this.props.view);
			}
		})();

		return (
			<AdminLayout title="Volumes &amp; mounts" breadcrumbs={[]}>
				<TabController
					tabs={[
						{
							url: volumesUrl(),
							title: 'Volumes',
						},
						{
							url: mountsUrl(),
							title: 'Mounts',
						},
						{
							url: volumesTopologyZonesUrl(),
							title: 'Topology & zones',
						},
						{
							url: volumesServiceUrl(),
							title: 'Service',
						},
						{
							url: volumesSmartUrl(),
							title: 'SMART',
						},
						{
							url: volumesIntegrityUrl(),
							title: 'Integrity',
						},
						{
							url: volumesReplicationUrl(),
							title: 'Replication status',
						},
					]}>
					{content}
				</TabController>
			</AdminLayout>
		);
	}

	private renderSmartView() {
		const [volumes, loadingOrError] = this.state.volumes.unwrap();

		if (!volumes || loadingOrError) {
			return loadingOrError;
		}

		return <SmartView volumes={volumes} />;
	}

	private renderServiceView() {
		const [volumes, loadingOrError] = this.state.volumes.unwrap();

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th>Label</th>
						<th>Description</th>
						<th>Serial number</th>
						<th>Manufactured</th>
						<th>Warranty ends</th>
					</tr>
				</thead>
				<tbody>
					{(volumes || []).map((vol) => {
						const manufactured = vol.Manufactured;
						const warrantyEnds = vol.WarrantyEnds;

						return (
							<tr key={vol.Id}>
								<td>{vol.Label}</td>
								<td>{vol.Description}</td>
								<td>
									{vol.SerialNumber}{' '}
									<CommandIcon
										command={VolumeSetSerialNumber(vol.Id, vol.SerialNumber)}
									/>
								</td>
								<td>
									{manufactured ? (
										<Timestamp ts={plainDateToDateTime(manufactured)} />
									) : (
										'-'
									)}{' '}
									<CommandIcon
										command={VolumeSetManufacturingDate(
											vol.Id,
											manufactured ? manufactured : ('' as dateRFC3339),
										)}
									/>
								</td>
								<td>
									{warrantyEnds ? (
										<Timestamp ts={plainDateToDateTime(warrantyEnds)} />
									) : (
										'-'
									)}{' '}
									<CommandIcon
										command={VolumeSetWarrantyEndDate(
											vol.Id,
											warrantyEnds ? warrantyEnds : ('' as dateRFC3339),
										)}
									/>
								</td>
							</tr>
						);
					})}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>{loadingOrError}</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private renderTopologyView() {
		const [volumes, mounts, loadingOrError] = Result.unwrap2(
			this.state.volumes,
			this.state.mounts,
		);

		if (!volumes || !mounts || loadingOrError) {
			return loadingOrError;
		}

		return <TopologyAndZonesView volumes={volumes} mounts={mounts} />;
	}

	private renderVolumes() {
		const [volumesMaybe, loadingOrError] = this.state.volumes.unwrap();
		const volumes = volumesMaybe || [];

		const blobCount = (vol: Volume) => thousandSeparate(vol.BlobCount);
		const free = (vol: Volume) => bytesToHumanReadable(vol.Quota - vol.BlobSizeTotal);
		const used = (vol: Volume) =>
			bytesToHumanReadable(vol.BlobSizeTotal) + ' / ' + bytesToHumanReadable(vol.Quota);
		const progressBar = (vol: Volume) => (
			<ProgressBar progress={(vol.BlobSizeTotal / vol.Quota) * 100} />
		);

		const toRow = (obj: Volume) => {
			return (
				<tr key={obj.Id}>
					<td title={`Uuid=${obj.Uuid} Id=${obj.Id}`}>{obj.Label}</td>
					<td>
						{volumeTechnologyBadge(obj.Technology)}
						&nbsp;
						{obj.Description}
						&nbsp;
						{obj.Notes && <Glyphicon icon="pencil" title={obj.Notes} />}
					</td>
					<td>{blobCount(obj)}</td>
					<td>{free(obj)}</td>
					<td>{used(obj)}</td>
					<td>{progressBar(obj)}</td>
					<td>
						<Dropdown>
							<CommandLink
								command={VolumeRename(obj.Id, obj.Label, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeChangeQuota(obj.Id, obj.Quota / 1024 / 1024, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeChangeDescription(obj.Id, obj.Description, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeChangeNotes(obj.Id, obj.Notes, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeSetTechnology(obj.Id, obj.Technology, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeMigrateData(
									obj.Id,
									{ To: volumeAutocomplete },
									{ disambiguation: obj.Label },
								)}
							/>
							<CommandLink
								command={VolumeMarkDataLost(obj.Id, {
									disambiguation: obj.Label,
									helpUrl: DocUrlLatest(DocRef.DocsUsingWhenADiskFailsIndexMd),
								})}
							/>
							<CommandLink
								command={VolumeRemoveQueuedReplications(obj.Id, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeDecommission(obj.Id, {
									disambiguation: obj.Label,
								})}
							/>
						</Dropdown>
					</td>
				</tr>
			);
		};

		const totals: Volume = volumes.reduce(
			(prev, vol: Volume) => {
				prev.BlobCount += vol.BlobCount;
				prev.Quota += vol.Quota;
				prev.BlobSizeTotal += vol.BlobSizeTotal;
				return prev;
			},
			{
				BlobCount: 0,
				Technology: VolumeTechnology.DiskHdd, // doesn't matter - not shown
				Quota: 0,
				BlobSizeTotal: 0,
				Description: '',
				Notes: '',
				Label: '',
				Uuid: '',
				SerialNumber: '',
				Smart: {
					Id: '',
					LatestReport: null,
				},
				Id: 0,
				Manufactured: null,
				WarrantyEnds: null,
				Zone: '',
				Topology: null,
				Decommissioned: null,
			},
		);

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th>Label</th>
						<th>Description</th>
						<th>Blob count</th>
						<th>Free</th>
						<th>Used</th>
						<th style={{ width: '220px' }}>
							<Info text="Quotas are soft quotas, and are currently not enforced. An alert will be raised if you go over the quota, though." />
						</th>
						<th />
					</tr>
				</thead>
				<tbody>{volumes.map(toRow)}</tbody>
				<tfoot>
					{loadingOrError ? (
						<tr>
							<td colSpan={99}>{loadingOrError}</td>
						</tr>
					) : null}
					<tr>
						<td />
						<td />
						<td>{blobCount(totals)}</td>
						<td>{free(totals)}</td>
						<td>{used(totals)}</td>
						<td>{progressBar(totals)}</td>
						<td />
					</tr>
				</tfoot>
			</table>
		);
	}

	private renderDecommissionedVolumes() {
		const [volumesDecommissioned, loadingOrError] = this.state.volumesDecommissioned.unwrap();

		if (!volumesDecommissioned || loadingOrError) {
			return loadingOrError;
		}

		if (!volumesDecommissioned.length) {
			return null;
		}

		// show recently decommissioned first
		volumesDecommissioned
			.sort((a, b) => (a.Decommissioned!.At < b.Decommissioned!.At ? -1 : 1))
			.reverse();

		return (
			<CollapsePanel
				heading={`${volumesDecommissioned.length} decommissioned volumes`}
				visualStyle="info">
				<table className={tableClassStripedHover}>
					<thead>
						<tr>
							<th>When</th>
							<th>Label</th>
							<th>Reason</th>
						</tr>
					</thead>
					<tbody>
						{volumesDecommissioned.map((vol) => (
							<tr key={vol.Uuid}>
								<td>
									<Timestamp ts={vol.Decommissioned!.At} />
								</td>
								<td>{vol.Label}</td>
								<td>{vol.Decommissioned!.Reason}</td>
							</tr>
						))}
					</tbody>
				</table>
			</CollapsePanel>
		);
	}

	private renderMounts() {
		const [mounts, volumes, nodes, loadingOrError] = Result.unwrap3(
			this.state.mounts,
			this.state.volumes,
			this.state.nodes,
		);

		if (!mounts || !volumes || !nodes || loadingOrError) {
			return loadingOrError;
		}

		const rows: JSX.Element[] = [];

		const mkRow = (vol: Volume, mount: VolumeMount | undefined, showVolume: boolean) => {
			const node = mount ? nodes.filter((n) => n.Id === mount.Node)[0] : undefined;

			const rowKey = vol.Uuid + (mount ? mount.Id : '');

			return (
				<tr key={rowKey}>
					<td>{mount && onlineBadge(mount.Online)}</td>
					<td>
						<span title={mount && 'MountId=' + mount.Id}>
							{showVolume && vol.Label}
						</span>
					</td>
					<td>{node && node.Name}</td>
					<td>{mount && mount.Driver}</td>
					<td>{mount && <SecretReveal secret={mount.DriverOpts} />}</td>
					<td>
						<Dropdown>
							<CommandLink
								command={VolumeMountLocal(vol.Id, {
									disambiguation: vol.Label,
									helpUrl: DocUrlLatest(DocRef.DocsStorageLocalFsIndexMd),
								})}
							/>
							<CommandLink
								command={VolumeMountGoogleDrive(vol.Id, {
									disambiguation: vol.Label,
									helpUrl: DocUrlLatest(DocRef.DocsStorageGoogledriveIndexMd),
									redirect: (createdRecordId): string => {
										if (createdRecordId === 'mounted-ok') {
											reloadCurrentPage();
										} else {
											if (
												confirm(
													'You´ll now be redirected to Google to authorize your account to access Google Drive',
												)
											) {
												window.open(createdRecordId, '_blank');
											}
										}
										return '';
									},
								})}
							/>
							<CommandLink
								command={VolumeMountS3(vol.Id, {
									disambiguation: vol.Label,
									helpUrl: DocUrlLatest(DocRef.DocsStorageS3IndexMd),
								})}
							/>
							{mount && (
								<CommandLink
									command={VolumeUnmount(mount.Id, {
										disambiguation: vol.Label,
									})}
								/>
							)}
						</Dropdown>
					</td>
				</tr>
			);
		};

		for (const vol of volumes) {
			const volumesMounts = mounts.filter((m) => m.Volume === vol.Id);

			if (!volumesMounts.length) {
				rows.push(mkRow(vol, undefined, true));
			} else {
				for (let i = 0; i < volumesMounts.length; i++) {
					rows.push(mkRow(vol, volumesMounts[i], i === 0));
				}
			}
		}

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th colSpan={2} style={{ textAlign: 'center' }}>
							Volume
						</th>
						<th colSpan={4} style={{ textAlign: 'center' }}>
							Mount
						</th>
					</tr>
					<tr>
						<th style={{ width: '1%' }} />
						<th></th>
						<th>Server</th>
						<th>Driver</th>
						<th>Driver options</th>
						<th />
					</tr>
				</thead>
				<tbody>{rows}</tbody>
			</table>
		);
	}

	private renderIvJobs() {
		const [ivJobs, volumes, loadingOrError] = Result.unwrap2(
			this.state.ivJobs,
			this.state.volumes,
		);

		if (!ivJobs || !volumes || loadingOrError) {
			return loadingOrError;
		}

		return (
			<IntegrityVerificationJobsView
				jobs={ivJobs}
				volumes={volumes}
				refresh={() => {
					this.loadIvJobs();
				}}
			/>
		);
	}

	private renderReplicationStatuses() {
		const [replicationStatuses, volumes, loadingOrError] = Result.unwrap2(
			this.state.replicationStatuses,
			this.state.volumes,
		);

		if (!replicationStatuses || !volumes || loadingOrError) {
			return loadingOrError;
		}

		interface ReplicationStatusAndVolume {
			status: ReplicationStatus;
			volume: Volume;
		}

		let statusesWithVolumes: ReplicationStatusAndVolume[] = [];

		// iterating from volumes' perspective to preserve meaningful volume
		// sort order from API
		volumes.forEach((volume) => {
			statusesWithVolumes = statusesWithVolumes.concat(
				replicationStatuses
					.filter((status) => status.VolumeId === volume.Id)
					.map((status) => {
						return { status, volume };
					}),
			);
		});

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th>Volume</th>
						<th>Progress</th>
					</tr>
				</thead>
				<tbody>
					{statusesWithVolumes.map((statusWithVolume) => {
						const [status, volume] = [statusWithVolume.status, statusWithVolume.volume];

						return (
							<tr key={status.VolumeId}>
								<td>{volume.Label}</td>
								<td>
									{status.Progress === 100 ? (
										'Realtime'
									) : (
										<ProgressBar progress={status.Progress} colour="warning" />
									)}
								</td>
							</tr>
						);
					})}
				</tbody>
			</table>
		);
	}

	private fetchData() {
		this.state.volumes.load(() => getVolumes());
		this.state.volumesDecommissioned.load(() => getDecommissionedVolumes());
		this.state.mounts.load(() => getVolumeMounts());
		this.state.nodes.load(() => getNodes());
		this.state.ivJobs.load(() => getIntegrityVerificationJobs());

		// refreshable, used from >1 places
		this.loadIvJobs();
		this.loadReplicationStatuses();
	}

	private loadIvJobs() {
		this.state.ivJobs.load(() => getIntegrityVerificationJobs());
	}

	private loadReplicationStatuses() {
		this.state.replicationStatuses.load(() => getReplicationStatuses());
	}
}
