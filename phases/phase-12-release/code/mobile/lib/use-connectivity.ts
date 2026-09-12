import * as Network from "expo-network";
import { useEffect, useState } from "react";

/**
 * Whether the venue network is currently usable (SRS 4.8).
 *
 * The door needs to know two things and they are not the same: whether there
 * is a connection at all, and whether the internet is actually reachable
 * through it. Venue Wi-Fi that has dropped its uplink still reports a
 * connection, so both have to be true before the app calls itself online and
 * tries to sync - otherwise it would fire a sync at a network that cannot carry
 * it and count the timeout as a failure.
 *
 * `isInternetReachable` is briefly undefined right after a change while the OS
 * re-probes; it is treated as "still online" so a momentary unknown does not
 * flip the door into offline mode mid-scan.
 */
export function useConnectivity(): { online: boolean; checked: boolean } {
  const [online, setOnline] = useState(true);
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    let active = true;

    const apply = (state: Network.NetworkState) => {
      if (!active) return;
      const reachable = state.isInternetReachable ?? true;
      setOnline(Boolean(state.isConnected) && reachable);
      setChecked(true);
    };

    void Network.getNetworkStateAsync().then(apply).catch(() => {
      if (active) setChecked(true);
    });

    const subscription = Network.addNetworkStateListener(apply);
    return () => {
      active = false;
      subscription.remove();
    };
  }, []);

  return { online, checked };
}
