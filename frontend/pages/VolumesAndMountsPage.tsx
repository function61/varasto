import { thousandSeparate } from 'component/numberformatter';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandIcon, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Loading } from 'f61ui/component/loading';
import { ProgressBar } from 'f61ui/component/progressbar';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import {
	VolumeChangeDescription,
	VolumeChangeQuota,
	VolumeCreate,
	VolumeMount2,
	VolumeUnmount,
} from 'generated/varastoserver_commands';
import { getNodes, getVolumeMounts, getVolumes } from 'generated/varastoserver_endpoints';
import { Node, Volume, VolumeMount } from 'generated/varastoserver_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

interface VolumesAndMountsPageState {
	volumes?: Volume[];
	mounts?: VolumeMount[];
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
			<AppDefaultLayout title="Volumes &amp; mounts" breadcrumbs={[]}>
				<Panel heading={volumesHeading}>{this.renderVolumes()}</Panel>

				<Panel heading="Mounts">{this.renderMounts()}</Panel>
			</AppDefaultLayout>
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
					<td>{obj.DriverOpts}</td>
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

	private async fetchData() {
		const [volumes, mounts, nodes] = await Promise.all([
			getVolumes(),
			getVolumeMounts(),
			getNodes(),
		]);

		this.setState({ volumes, mounts, nodes });
	}
}
