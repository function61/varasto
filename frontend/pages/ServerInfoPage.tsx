import { Result } from 'component/result';
import {
	changeSensitivity,
	getMaxSensitivityFromLocalStorage,
	Sensitivity,
	sensitivityLabel,
} from 'component/sensitivity';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { Timestamp } from 'f61ui/component/timestamp';
import { unrecognizedValue } from 'f61ui/utils';
import { getHealth, getServerInfo } from 'generated/stoserver/stoservertypes_endpoints';
import { Health, HealthStatus, ServerInfo } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface ServerInfoPageState {
	serverInfo: Result<ServerInfo>;
	health: Result<Health>;
	currSens: Sensitivity;
}

export default class ServerInfoPage extends React.Component<{}, ServerInfoPageState> {
	state: ServerInfoPageState = {
		serverInfo: new Result<ServerInfo>((_) => {
			this.setState({ serverInfo: _ });
		}),
		health: new Result<Health>((_) => {
			this.setState({ health: _ });
		}),
		currSens: getMaxSensitivityFromLocalStorage(),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		return (
			<SettingsLayout title="Server info &amp; health" breadcrumbs={[]}>
				<Panel heading="Server info">{this.renderInfo()}</Panel>
				<Panel heading="Health">{this.renderHealth()}</Panel>
				<Panel heading="Sensitivity">{this.renderSensitivitySelector()}</Panel>
			</SettingsLayout>
		);
	}
	private renderInfo() {
		const [serverInfo, loadingOrError] = this.state.serverInfo.unwrap();

		interface Item {
			h: string;
			t: React.ReactNode;
		}

		const items: Item[] = serverInfo
			? [
					{ h: 'Varasto version', t: serverInfo.AppVersion },
					{ h: 'Varasto uptime', t: <Timestamp ts={serverInfo.StartedAt} /> },
					{ h: 'Database size', t: bytesToHumanReadable(serverInfo.DatabaseSize) },
					{ h: 'Go version', t: serverInfo.GoVersion },
					{
						h: 'Server OS / arch',
						t: `${serverInfo.ServerOs} / ${serverInfo.ServerArch}`,
					},
					{ h: 'Process ID', t: serverInfo.ProcessId },
					{ h: 'CPU count', t: serverInfo.CpuCount.toString() },
					{ h: 'Goroutines', t: serverInfo.Goroutines.toString() },
					{ h: 'Heap memory', t: bytesToHumanReadable(serverInfo.HeapBytes) },
			  ]
			: [];

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
						<td colSpan={99}>{loadingOrError}</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private renderHealth() {
		const [health, loadingOrError] = this.state.health.unwrap();

		const rows: JSX.Element[] = [];

		const pushHealthNodeAsRow = (node: Health, indentLevel: number) => {
			rows.push(
				<tr>
					<td>{healthStatusToIcon(node.Health)}</td>
					<td style={{ paddingLeft: indentLevel * 32 + 'px' }}>{node.Title}</td>
					<td>{node.Details}</td>
				</tr>,
			);

			node.Children.forEach((childHealth) => {
				pushHealthNodeAsRow(childHealth, indentLevel + 1);
			});
		};

		if (health) {
			pushHealthNodeAsRow(health, 0);
		}

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th />
						<th>Title</th>
						<th>Details</th>
					</tr>
				</thead>
				<tbody>{rows}</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>{loadingOrError}</td>
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

	private fetchData() {
		this.state.serverInfo.load(() => getServerInfo());
		this.state.health.load(() => getHealth());
	}
}

function healthStatusToIcon(input: HealthStatus): JSX.Element {
	switch (input) {
		case HealthStatus.Fail:
			return (
				<span className="alert alert-danger">
					<span className="glyphicon glyphicon-fire" />
				</span>
			);
		case HealthStatus.Warn:
			return (
				<span className="alert alert-warning">
					<span className="glyphicon glyphicon-warning-sign" />
				</span>
			);
		case HealthStatus.Pass:
			return (
				<span className="alert alert-success">
					<span className="glyphicon glyphicon-ok" />
				</span>
			);
		default:
			throw unrecognizedValue(input);
	}
}
