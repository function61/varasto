import { DocLink } from 'component/doclink';
import { DangerLabel, SuccessLabel, WarningLabel } from 'component/labels';
import { thousandSeparate } from 'component/numberformatter';
import { RefreshButton } from 'component/refreshbutton';
import { Result } from 'component/result';
import { TabController } from 'component/tabcontroller';
import { InfoAlert } from 'f61ui/component/alerts';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandButton, CommandIcon, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Info } from 'f61ui/component/info';
import { ProgressBar } from 'f61ui/component/progressbar';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import { plainDateToDateTime } from 'f61ui/types';
import { unrecognizedValue } from 'f61ui/utils';
import {
	IntegrityverificationjobResume,
	IntegrityverificationjobStop,
	NodeSmartScan,
	VolumeChangeDescription,
	VolumeChangeQuota,
	VolumeCreate,
	VolumeMarkDataLost,
	VolumeMigrateData,
	VolumeMount2,
	VolumeSetManufacturingDate,
	VolumeSetSerialNumber,
	VolumeSetTechnology,
	VolumeSetTopology,
	VolumeSetWarrantyEndDate,
	VolumeSmartSetId,
	VolumeUnmount,
	VolumeVerifyIntegrity,
} from 'generated/stoserver/stoservertypes_commands';
import {
	getIntegrityVerificationJobs,
	getNodes,
	getReplicationStatuses,
	getVolumeMounts,
	getVolumes,
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
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';
import { volumesAndMountsRoute } from 'routes';

interface VolumesAndMountsPageProps {
	view: string;
}

interface VolumesAndMountsPageState {
	volumes: Result<Volume[]>;
	mounts: Result<VolumeMount[]>;
	ivJobs: Result<IntegrityVerificationJob[]>;
	nodes: Result<Node[]>;
	replicationStatuses: Result<ReplicationStatus[]>;
}

interface Enclosure {
	name: string;
	bays: Array<{
		slot: number;
		volume: Volume | null;
	}>;
}

export default class VolumesAndMountsPage extends React.Component<
	VolumesAndMountsPageProps,
	VolumesAndMountsPageState
> {
	state: VolumesAndMountsPageState = {
		volumes: new Result<Volume[]>((_) => {
			this.setState({ volumes: _ });
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
							<Panel heading={volumesHeading()}>{this.renderVolumes()}</Panel>

							<Panel heading="Data integrity verification jobs">
								{this.renderIvJobs()}
							</Panel>
						</div>
					);
				case 'topology':
					return (
						<Panel
							heading={
								<div>
									Topology view{' '}
									<Info text="If you have a lot of disks, it's great to know where they're physically located, so if you need to detach a disk you know to detact the right one." />
								</div>
							}>
							{this.renderTopologyView()}
						</Panel>
					);
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
									Replication{' '}
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
									SMART{' '}
									<DocLink doc={DocRef.DocsGuideSettingUpSmartMonitoringMd} />
								</div>
							}>
							{this.renderSmartView()}
						</Panel>
					);
				case '':
					return (
						<div>
							<Panel heading={volumesHeading()}>{this.renderVolumes()}</Panel>

							<Panel heading="Mounts">{this.renderMounts()}</Panel>
						</div>
					);
				default:
					throw new Error('unknown view');
			}
		})();

		return (
			<SettingsLayout title="Volumes &amp; mounts" breadcrumbs={[]}>
				<TabController
					tabs={[
						{
							url: volumesAndMountsRoute.buildUrl({
								view: '',
							}),
							title: 'Volumes & mounts',
						},
						{
							url: volumesAndMountsRoute.buildUrl({
								view: 'topology',
							}),
							title: 'Topology',
						},
						{
							url: volumesAndMountsRoute.buildUrl({
								view: 'service',
							}),
							title: 'Service',
						},
						{
							url: volumesAndMountsRoute.buildUrl({
								view: 'smart',
							}),
							title: 'SMART',
						},
						{
							url: volumesAndMountsRoute.buildUrl({
								view: 'integrity',
							}),
							title: 'Integrity',
						},
						{
							url: volumesAndMountsRoute.buildUrl({
								view: 'replicationStatuses',
							}),
							title: 'Replication',
						},
					]}>
					{content}
				</TabController>
			</SettingsLayout>
		);
	}

	private renderSmartView() {
		const [volumes, loadingOrError] = this.state.volumes.unwrap();

		const volumesWithSmart = (volumes || []).filter((vol) => !!vol.Smart.LatestReport);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Passed</th>
						<th>Label</th>
						<th>Description</th>
						<th>Reported</th>
						<th>Temperature</th>
						<th>PowerCycleCount</th>
						<th>PowerOnTime</th>
					</tr>
				</thead>
				<tbody>
					{volumesWithSmart.map((vol) => {
						const smart = vol.Smart.LatestReport!;

						return (
							<tr key={vol.Id}>
								<td>
									{smart.Passed ? (
										<SuccessLabel title="Pass">✓</SuccessLabel>
									) : (
										<DangerLabel title="Fail">❌</DangerLabel>
									)}
								</td>
								<td>{vol.Label}</td>
								<td>{vol.Description}</td>
								<td>
									<Timestamp ts={smart.Time} />
								</td>
								<td>
									{smart.Temperature
										? smart.Temperature.toString() + ' °C'
										: null}
								</td>
								<td>
									{smart.PowerCycleCount
										? thousandSeparate(smart.PowerCycleCount)
										: null}
								</td>
								<td>
									{smart.PowerOnTime ? thousandSeparate(smart.PowerOnTime) : null}
								</td>
							</tr>
						);
					})}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							<div>{loadingOrError}</div>
							{volumesWithSmart.length === 0 ? (
								<div>
									<InfoAlert>
										No SMART-reporting volumes found. Read docs first:{' '}
										<DocLink doc={DocRef.DocsGuideSettingUpSmartMonitoringMd} />
									</InfoAlert>
								</div>
							) : null}
							<CommandButton command={NodeSmartScan()} />
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private renderServiceView() {
		const [volumes, loadingOrError] = this.state.volumes.unwrap();

		return (
			<table className="table table-striped table-hover">
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
											manufactured ? manufactured : ('' as any),
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
											warrantyEnds ? warrantyEnds : ('' as any),
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

		const isOnline = (volId: number): boolean => {
			const matchingMount = mounts.filter((m) => m.Volume === volId);

			return matchingMount.length > 0 ? matchingMount[0].Online : false;
		};

		const enclosures: Enclosure[] = [];

		const addEnclosure = (name: string) => {
			const enc = {
				name,
				bays: [],
			};
			enclosures.push(enc);
			return enc;
		};

		volumes.forEach((volume) => {
			const enclosureName = volume.Topology ? volume.Topology.Enclosure : '(No enclosure)';

			const matches = enclosures.filter((enc) => enc.name === enclosureName);

			const enclosure = matches.length === 1 ? matches[0] : addEnclosure(enclosureName);

			enclosure.bays.push({
				slot: volume.Topology ? volume.Topology.Slot : 0,
				volume,
			});
		});

		enclosures.forEach((enclosure) => {
			const maxSlot = enclosure.bays.reduce((acc, curr) => Math.max(acc, curr.slot), 0);

			for (let i = 1; i < maxSlot; i++) {
				if (enclosure.bays.filter((bay) => bay.slot === i).length === 0) {
					enclosure.bays.push({ slot: i, volume: null }); // unpopulated slot
				}
			}

			enclosure.bays.sort((a, b) => (a.slot < b.slot ? -1 : 1));
		});

		enclosures.sort((a, b) => (a.name < b.name ? -1 : 1));

		return (
			<div className="row">
				{enclosures.map((enclosure) => (
					<div className="col-md-4">
						<table className="table table-bordered table-striped table-hover">
							<thead>
								<tr>
									<th />
									<th />
									<th>{enclosure.name}</th>
								</tr>
							</thead>
							<tbody>
								{enclosure.bays.map((bay) => (
									<tr>
										<td>{bay.slot}</td>
										<td>
											{bay.volume ? onlineBadge(isOnline(bay.volume.Id)) : ''}
										</td>
										<td>
											{bay.volume ? bay.volume.Label : ''}
											{bay.volume ? (
												<CommandIcon
													command={VolumeSetTopology(
														bay.volume.Id,
														bay.volume.Topology
															? bay.volume.Topology.Enclosure
															: '',
														bay.volume.Topology
															? bay.volume.Topology.Slot
															: 0,
													)}
												/>
											) : null}
										</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				))}
			</div>
		);
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
						<span className="label label-default">
							{volumeTechnologyToDisplay(obj.Technology)}
						</span>{' '}
						{obj.Description}
					</td>
					<td>{blobCount(obj)}</td>
					<td>{free(obj)}</td>
					<td>{used(obj)}</td>
					<td>{progressBar(obj)}</td>
					<td>
						<Dropdown>
							<CommandLink
								command={VolumeMount2(obj.Id, { disambiguation: obj.Label })}
							/>
							<CommandLink
								command={VolumeChangeQuota(obj.Id, obj.Quota / 1024 / 1024, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeVerifyIntegrity(obj.Id, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeChangeDescription(obj.Id, obj.Description, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeSetTechnology(obj.Id, obj.Technology, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeSmartSetId(obj.Id, obj.Smart.Id, {
									disambiguation: obj.Label,
								})}
							/>
							<CommandLink
								command={VolumeMigrateData(obj.Id, { disambiguation: obj.Label })}
							/>
							<CommandLink
								command={VolumeMarkDataLost(obj.Id, { disambiguation: obj.Label })}
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
				Topology: null,
			},
		);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Label</th>
						<th>Description</th>
						<th>Blob count</th>
						<th>Free</th>
						<th>Used</th>
						<th style={{ width: '220px' }} />
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

	private renderMounts() {
		const [mounts, volumes, nodes, loadingOrError] = Result.unwrap3(
			this.state.mounts,
			this.state.volumes,
			this.state.nodes,
		);

		if (!mounts || !volumes || !nodes || loadingOrError) {
			return loadingOrError;
		}

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th style={{ width: '1%' }} />
						<th>Volume</th>
						<th>Node</th>
						<th>Driver</th>
						<th>DriverOpts</th>
						<th />
					</tr>
				</thead>
				<tbody>
					{mounts.map((mount) => {
						const volume = volumes.filter((vol) => vol.Id === mount.Volume);
						const node = nodes.filter((nd) => nd.Id === mount.Node);

						const volumeName = volume.length === 1 ? volume[0].Label : '(error)';
						const nodeName = node.length === 1 ? node[0].Name : '(error)';

						return (
							<tr key={mount.Id}>
								<td>{onlineBadge(mount.Online)}</td>
								<td>
									<span title={`MountId=${mount.Id}`}>{volumeName}</span>
								</td>
								<td>{nodeName}</td>
								<td>{mount.Driver}</td>
								<td>
									<SecretReveal secret={mount.DriverOpts} />
								</td>
								<td>
									<CommandIcon
										command={VolumeUnmount(mount.Id, {
											disambiguation: volumeName,
										})}
									/>
								</td>
							</tr>
						);
					})}
				</tbody>
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

		/*
			stopped = !isCompleted AND !running
			running = !isCompleted AND running
			pass = isCompleted AND errors == 0
			fail = isCompleted AND errors > 0
		*/
		const jobStatus = (obj: IntegrityVerificationJob): React.ReactNode => {
			const completed = obj.Completed;

			if (completed === null) {
				if (!obj.Running) {
					return <WarningLabel>Stopped</WarningLabel>;
				}

				// since the blobref is a SHA256, and its properties is uniform random distribution,
				// and since our b-tree based database table scans are alphabetical order, we
				// can deduce progress of scan by just looking at four first hexits:
				//
				// 0000 =>   0 %
				// 8000 =>  50 %
				// ffff => 100 %
				const lastCompletedBlobRefFourFirstHexits = obj.LastCompletedBlobRef.substr(0, 4);

				const progress = (parseInt(lastCompletedBlobRefFourFirstHexits, 16) / 65535) * 100;

				return <ProgressBar progress={progress} />;
			}

			if (obj.ErrorsFound > 0) {
				return <DangerLabel>Failed</DangerLabel>;
			}

			return (
				<SuccessLabel>
					Pass <Timestamp ts={completed} />
				</SuccessLabel>
			);
		};

		const toRow = (obj: IntegrityVerificationJob) => {
			const volume = volumes.filter((vol) => vol.Id === obj.VolumeId);
			const volumeName = volume.length === 1 ? volume[0].Label : '(error)';

			return (
				<tr key={obj.Id}>
					<td>{jobStatus(obj)}</td>
					<td>{volumeName}</td>
					<td title={obj.Id}>
						<Timestamp ts={obj.Created} />
					</td>
					<td>{bytesToHumanReadable(obj.BytesScanned)}</td>
					<td title={'Errors found: ' + thousandSeparate(obj.ErrorsFound)}>
						{obj.Report}
					</td>
					<td>
						<Dropdown>
							<CommandLink command={IntegrityverificationjobResume(obj.Id)} />
							<CommandLink command={IntegrityverificationjobStop(obj.Id)} />
						</Dropdown>
					</td>
				</tr>
			);
		};

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th />
						<th>Volume</th>
						<th>Created</th>
						<th>Scanned</th>
						<th>Report</th>
						<th style={{ width: '1%' }} />
					</tr>
				</thead>
				<tbody>{ivJobs.map(toRow)}</tbody>
			</table>
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

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Volume</th>
						<th>
							Progress{' '}
							<Info text="Doesn't update in realtime. If you don't see a change you're expecting, wait a few minutes." />
						</th>
					</tr>
				</thead>
				<tbody>
					{replicationStatuses.map((status) => {
						const volume = volumes.filter((vol) => vol.Id === status.VolumeId);
						const volumeName = volume.length === 1 ? volume[0].Label : '(error)';

						return (
							<tr key={status.VolumeId}>
								<td>{volumeName}</td>
								<td>
									<ProgressBar
										progress={status.Progress}
										colour={status.Progress < 100 ? 'warning' : undefined}
									/>
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
		this.state.mounts.load(() => getVolumeMounts());
		this.state.nodes.load(() => getNodes());
		this.state.ivJobs.load(() => getIntegrityVerificationJobs());

		this.loadReplicationStatuses();
	}

	// used from >1 places
	private loadReplicationStatuses() {
		this.state.replicationStatuses.load(() => getReplicationStatuses());
	}
}

function volumeTechnologyToDisplay(tech: VolumeTechnology): string {
	switch (tech) {
		case VolumeTechnology.DiskHdd:
			return 'HDD';
		case VolumeTechnology.DiskSsd:
			return 'SSD';
		case VolumeTechnology.Cloud:
			return '☁';
		default:
			throw unrecognizedValue(tech);
	}
}

function onlineBadge(online: boolean): React.ReactNode {
	return online ? (
		<SuccessLabel title="Online">
			<span className="glyphicon glyphicon-off" />
		</SuccessLabel>
	) : (
		<DangerLabel title="Offline">
			<span className="glyphicon glyphicon-off" />
		</DangerLabel>
	);
}
