import { thousandSeparate } from 'component/numberformatter';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandIcon, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Loading } from 'f61ui/component/loading';
import { ProgressBar } from 'f61ui/component/progressbar';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { Timestamp } from 'f61ui/component/timestamp';
import { jsxChildType } from 'f61ui/types';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import {
	IntegrityverificationjobResume,
	IntegrityverificationjobStop,
	VolumeChangeDescription,
	VolumeChangeQuota,
	VolumeCreate,
	VolumeMount2,
	VolumeUnmount,
	VolumeVerifyIntegrity,
} from 'generated/stoserver/stoservertypes_commands';
import {
	getIntegrityVerificationJobs,
	getNodes,
	getVolumeMounts,
	getVolumes,
} from 'generated/stoserver/stoservertypes_endpoints';
import {
	IntegrityVerificationJob,
	Node,
	Volume,
	VolumeMount,
} from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface VolumesAndMountsPageState {
	volumes?: Volume[];
	mounts?: VolumeMount[];
	ivJobs?: IntegrityVerificationJob[];
	nodes?: Node[];
}

export default class VolumesAndMountsPage extends React.Component<{}, VolumesAndMountsPageState> {
	state: VolumesAndMountsPageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		const volumesHeading = (
			<div>
				Volumes &nbsp;
				<Dropdown>
					<CommandLink command={VolumeCreate()} />
				</Dropdown>
			</div>
		);

		return (
			<SettingsLayout title="Volumes &amp; mounts" breadcrumbs={[]}>
				<Panel heading={volumesHeading}>{this.renderVolumes()}</Panel>

				<Panel heading="Mounts">{this.renderMounts()}</Panel>

				<Panel heading="Data integrity verification jobs">{this.renderIvJobs()}</Panel>
			</SettingsLayout>
		);
	}

	private renderVolumes() {
		const volumes = this.state.volumes;

		if (!volumes) {
			return <Loading />;
		}

		const blobCount = (vol: Volume) => thousandSeparate(vol.BlobCount);
		const free = (vol: Volume) => bytesToHumanReadable(vol.Quota - vol.BlobSizeTotal);
		const used = (vol: Volume) =>
			bytesToHumanReadable(vol.BlobSizeTotal) + ' / ' + bytesToHumanReadable(vol.Quota);
		const progressBar = (vol: Volume) => (
			<ProgressBar progress={(vol.BlobSizeTotal / vol.Quota) * 100} />
		);

		const toRow = (obj: Volume) => {
			// TODO: this is a stupid heuristic
			const tb = 1024 * 1024 * 1024 * 1024;
			let techName = obj.Quota < 1 * tb ? 'SSD' : 'HDD';

			// email address? => maybe cloud account
			if (obj.Description.indexOf('@') !== -1) {
				techName = '☁';
			}

			const techTag = <span className="label label-default">{techName}</span>;

			return (
				<tr key={obj.Id}>
					<td title={`Uuid=${obj.Uuid} Id=${obj.Id}`}>{obj.Label}</td>
					<td>
						{techTag} {obj.Description}
					</td>
					<td>{blobCount(obj)}</td>
					<td>{free(obj)}</td>
					<td>{used(obj)}</td>
					<td>{progressBar(obj)}</td>
					<td>
						<Dropdown>
							<CommandLink command={VolumeMount2(obj.Id)} />
							<CommandLink command={VolumeChangeQuota(obj.Id, obj.Quota)} />
							<CommandLink command={VolumeVerifyIntegrity(obj.Id)} />
							<CommandLink
								command={VolumeChangeDescription(obj.Id, obj.Description)}
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
				Quota: 0,
				BlobSizeTotal: 0,
				Description: '',
				Label: '',
				Uuid: '',
				Id: 0,
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
		const mounts = this.state.mounts;
		const volumes = this.state.volumes;
		const nodes = this.state.nodes;

		if (!mounts || !volumes || !nodes) {
			return <Loading />;
		}

		const toRow = (obj: VolumeMount) => {
			const onlineBadge = obj.Online ? (
				<span className="label label-success">Online</span>
			) : (
				<span className="label label-danger">Offline</span>
			);

			const volume = volumes.filter((vol) => vol.Id === obj.Volume);
			const node = nodes.filter((nd) => nd.Id === obj.Node);

			const volumeName = volume.length === 1 ? volume[0].Label : '(error)';
			const nodeName = node.length === 1 ? node[0].Name : '(error)';

			return (
				<tr key={obj.Id}>
					<td>{onlineBadge}</td>
					<td>
						<span title={`MountId=${obj.Id}`}>{volumeName}</span>
					</td>
					<td>{nodeName}</td>
					<td>{obj.Driver}</td>
					<td>
						<SecretReveal secret={obj.DriverOpts} />
					</td>
					<td>
						<CommandIcon command={VolumeUnmount(obj.Id)} />
					</td>
				</tr>
			);
		};

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
				<tbody>{mounts.map(toRow)}</tbody>
			</table>
		);
	}

	private renderIvJobs() {
		const ivJobs = this.state.ivJobs;
		const volumes = this.state.volumes;

		if (!ivJobs || !volumes) {
			return <Loading />;
		}

		/*
			stopped = !isCompleted AND !running
			running = !isCompleted AND running
			pass = isCompleted AND errors == 0
			fail = isCompleted AND errors > 0
		*/
		const jobStatus = (obj: IntegrityVerificationJob): jsxChildType => {
			const completed = obj.Completed;

			if (completed === null) {
				if (!obj.Running) {
					return <span className="label label-warning">Stopped</span>;
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
				return <span className="label label-danger">Failed</span>;
			}

			return (
				<span className="label label-success">
					Pass <Timestamp ts={completed} />
				</span>
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

	private async fetchData() {
		const [volumes, mounts, nodes, ivJobs] = await Promise.all([
			getVolumes(),
			getVolumeMounts(),
			getNodes(),
			getIntegrityVerificationJobs(),
		]);

		this.setState({ volumes, mounts, nodes, ivJobs });
	}
}
