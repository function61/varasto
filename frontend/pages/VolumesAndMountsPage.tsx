import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { Loading } from 'f61ui/component/loading';
import { ProgressBar } from 'f61ui/component/progressbar';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { getVolumeMounts, getVolumes } from 'generated/bupserver_endpoints';
import { Volume, VolumeMount } from 'generated/bupserver_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

interface VolumesAndMountsPageState {
	volumes?: Volume[];
	mounts?: VolumeMount[];
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

		const toRow = (obj: Volume) => (
			<tr key={obj.Id}>
				<td>{obj.Id}</td>
				<td>{obj.Uuid}</td>
				<td>{obj.Label}</td>
				<td>{bytesToHumanReadable(obj.BlobSizeTotal)}</td>
				<td>{obj.BlobCount}</td>
				<td>
					<ProgressBar progress={(obj.BlobSizeTotal / obj.Quota) * 100} />
				</td>
			</tr>
		);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Id</th>
						<th>Uuid</th>
						<th>Label</th>
						<th>Blob total size</th>
						<th>Blob count</th>
						<th>Quota used</th>
					</tr>
				</thead>
				<tbody>{volumes.map(toRow)}</tbody>
			</table>
		);
	}

	private renderMounts() {
		const mounts = this.state.mounts;

		if (!mounts) {
			return <Loading />;
		}

		const toRow = (obj: VolumeMount) => (
			<tr key={obj.Id}>
				<td>{obj.Id}</td>
				<td>{obj.Volume}</td>
				<td>{obj.Node}</td>
				<td>{obj.Driver}</td>
				<td>{obj.DriverOpts}</td>
			</tr>
		);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Id</th>
						<th>Volume</th>
						<th>Node</th>
						<th>Driver</th>
						<th>DriverOpts</th>
					</tr>
				</thead>
				<tbody>{mounts.map(toRow)}</tbody>
			</table>
		);
	}

	private async fetchData() {
		const volumes = await getVolumes();
		const mounts = await getVolumeMounts();

		this.setState({ volumes, mounts });
	}
}
