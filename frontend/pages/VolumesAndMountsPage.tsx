import { thousandSeparate } from 'component/numberformatter';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandButton, CommandIcon, CommandLink } from 'f61ui/component/CommandButton';
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
		return (
			<AppDefaultLayout title="Volumes &amp; mounts" breadcrumbs={[]}>
				<Panel heading="Volumes">{this.renderVolumes()}</Panel>

				<Panel heading="Mounts">{this.renderMounts()}</Panel>
			</AppDefaultLayout>
		);
	}

	private renderVolumes() {
		const volumes = this.state.volumes;

		if (!volumes) {
			return <Loading />;
		}

		const toRow = (obj: Volume) => {
			// TODO: this is a stupid heuristic
			const tb = 1024 * 1024 * 1024 * 1024;
			const techName = obj.Quota < 1 * tb ? 'SSD' : 'HDD';

			const techTag = <span className="label label-default">{techName}</span>;

			return (
				<tr key={obj.Id}>
					<td title={`Uuid=${obj.Uuid} Id=${obj.Id}`}>{obj.Label}</td>
					<td>
						{techTag} {obj.Description}
					</td>
					<td>{thousandSeparate(obj.BlobCount)}</td>
					<td>
						{bytesToHumanReadable(obj.BlobSizeTotal)} /{' '}
						{bytesToHumanReadable(obj.Quota)}
					</td>
					<td>
						<ProgressBar progress={(obj.BlobSizeTotal / obj.Quota) * 100} />
					</td>
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

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Label</th>
						<th>Make/model</th>
						<th>Blob count</th>
						<th>Usage</th>
						<th style={{ width: '220px' }} />
						<th />
					</tr>
				</thead>
				<tbody>{volumes.map(toRow)}</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							<CommandButton command={VolumeCreate()} />
						</td>
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
					<td>
						<span title={`MountId=${obj.Id}`} className="margin-right">
							{volumeName}
						</span>
						&nbsp;
						{onlineBadge}
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
