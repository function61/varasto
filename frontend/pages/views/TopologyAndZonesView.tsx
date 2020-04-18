import {
	DangerLabel,
	Glyphicon,
	Panel,
	SuccessLabel,
	Well,
	tableClassStripedHoverBordered,
} from 'f61ui/component/bootstrap';
import { WarningAlert } from 'f61ui/component/alerts';
import { CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Info } from 'f61ui/component/info';
import { Volume, VolumeMount } from 'generated/stoserver/stoservertypes_types';
import { VolumeSetTopology, VolumeChangeZone } from 'generated/stoserver/stoservertypes_commands';
import * as React from 'react';

interface Enclosure {
	name: string;
	bays: {
		slot: number;
		volume: Volume | null;
	}[];
}

interface TopologyAndZonesViewProps {
	volumes: Volume[];
	mounts: VolumeMount[];
}

export default class TopologyAndZonesView extends React.Component<TopologyAndZonesViewProps, {}> {
	render() {
		const uniqueZones: string[] = [];

		for (const vol of this.props.volumes) {
			if (uniqueZones.indexOf(vol.Zone) === -1) {
				uniqueZones.push(vol.Zone);
			}
		}

		uniqueZones.sort();

		return (
			<div>
				<Well>
					Your disk topology{' '}
					<Info text="If you have a lot of disks, it's great to know where they're physically located, so if you need to detach a disk you know to detact the right one." />{' '}
					and zones{' '}
					<Info text="Physically separate location for your volumes regarding fire/water/power/network connectivity safety." />
					.
				</Well>

				{uniqueZones.length < 2 && (
					<WarningAlert>
						Looks like your volumes exist in one zone only. That means your data is not
						safe from fire/water/other damage or available on power loss or network
						connectivity issues. 🔥 🌊 🔌
					</WarningAlert>
				)}

				{uniqueZones.map((zone) => {
					const volumesForZone = this.props.volumes.filter((v) => v.Zone === zone);

					return (
						<Panel key={zone} heading={'Zone: ' + zone}>
							{this.renderZoneTopologyView(volumesForZone)}
						</Panel>
					);
				})}
			</div>
		);
	}

	private renderZoneTopologyView(volumesForZone: Volume[]) {
		const isOnline = (volId: number): boolean => {
			const matchingMount = this.props.mounts.filter((m) => m.Volume === volId);

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

		volumesForZone.forEach((volume) => {
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

		return enclosures.map((enclosure) => (
			<div key={enclosure.name} className="col-md-4">
				<table className={tableClassStripedHoverBordered}>
					<thead>
						<tr>
							<th />
							<th />
							<th>{enclosure.name}</th>
							<th />
						</tr>
					</thead>
					<tbody>
						{enclosure.bays.map((bay) => {
							const vol = bay.volume;
							if (!vol) {
								return (
									<tr>
										<td>{bay.slot}</td>
										<td></td>
										<td></td>
										<td></td>
									</tr>
								);
							}

							return (
								<tr>
									<td>{bay.slot}</td>
									<td>{onlineBadge(isOnline(vol.Id))}</td>
									<td>{vol.Label}</td>
									<td>
										<Dropdown>
											<CommandLink
												command={VolumeSetTopology(
													vol.Id,
													vol.Topology ? vol.Topology.Enclosure : '',
													vol.Topology ? vol.Topology.Slot : 0,
												)}
											/>
											<CommandLink
												command={VolumeChangeZone(vol.Id, vol.Zone)}
											/>
										</Dropdown>
									</td>
								</tr>
							);
						})}
					</tbody>
				</table>
			</div>
		));
	}
}

export function onlineBadge(online: boolean): React.ReactNode {
	return online ? (
		<SuccessLabel title="Online">
			<Glyphicon icon="off" />
		</SuccessLabel>
	) : (
		<DangerLabel title="Offline">
			<Glyphicon icon="off" />
		</DangerLabel>
	);
}
