import {
	changeSensitivity,
	getMaxSensitivityFromLocalStorage,
	Sensitivity,
	sensitivityLabel,
} from 'component/sensitivity';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandButton } from 'f61ui/component/CommandButton';
import { Loading } from 'f61ui/component/loading';
import { Timestamp } from 'f61ui/component/timestamp';
import { jsxChildType } from 'f61ui/types';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { DatabaseBackup } from 'generated/stoserver/stoservertypes_commands';
import { getServerInfo } from 'generated/stoserver/stoservertypes_endpoints';
import { ServerInfo } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface ServerInfoPageState {
	serverInfo?: ServerInfo;
	currSens: Sensitivity;
}

export default class ServerInfoPage extends React.Component<{}, ServerInfoPageState> {
	state: ServerInfoPageState = { currSens: getMaxSensitivityFromLocalStorage() };

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<SettingsLayout title="Server info" breadcrumbs={[]}>
				<Panel heading="Server info">{this.renderInfo()}</Panel>
				<Panel heading="Sensitivity">{this.renderSensitivitySelector()}</Panel>
			</SettingsLayout>
		);
	}

	private renderInfo() {
		const serverInfo = this.state.serverInfo;

		if (!serverInfo) {
			return <Loading />;
		}

		interface Item {
			h: string;
			t: jsxChildType;
		}

		const items: Item[] = [
			{ h: 'Varasto version', t: serverInfo.AppVersion },
			{ h: 'Varasto uptime', t: <Timestamp ts={serverInfo.StartedAt} /> },
			{ h: 'Database size', t: bytesToHumanReadable(serverInfo.DatabaseSize) },
			{ h: 'Go version', t: serverInfo.GoVersion },
			{ h: 'Server OS / arch', t: `${serverInfo.ServerOs} / ${serverInfo.ServerArch}` },
			{ h: 'CPU count', t: serverInfo.CpuCount.toString() },
			{ h: 'Goroutines', t: serverInfo.Goroutines.toString() },
			{ h: 'Heap memory', t: bytesToHumanReadable(serverInfo.HeapBytes) },
		];

		return (
			<table className="table table-striped table-hover">
				<tbody>
					{items.map((item) => (
						<tr key={item.h}>
							<th>{item.h}</th>
							<td>{item.t}</td>
						</tr>
					))}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							<CommandButton command={DatabaseBackup()} />
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private renderSensitivitySelector() {
		const sensitivityRadioChange = (e: React.ChangeEvent<HTMLInputElement>) => {
			changeSensitivity(+e.target.value);
			this.setState({ currSens: getMaxSensitivityFromLocalStorage() });
		};

		const oneSensitivityRadio = (sens: Sensitivity) => (
			<div key={sens}>
				<label>
					<input
						type="radio"
						name="changeSensitivityRadio"
						onChange={sensitivityRadioChange}
						value={sens}
						checked={sens === this.state.currSens}
					/>{' '}
					{sensitivityLabel(sens)}
				</label>
			</div>
		);

		return (
			<div>
				{oneSensitivityRadio(Sensitivity.FamilyFriendly)}
				{oneSensitivityRadio(Sensitivity.Sensitive)}
				{oneSensitivityRadio(Sensitivity.MyEyesOnly)}
			</div>
		);
	}

	private async fetchData() {
		const serverInfo = await getServerInfo();

		this.setState({ serverInfo });
	}
}
